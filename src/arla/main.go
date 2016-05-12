package main

import (
	"arla/mutationstore"
	"arla/querystore"
	"arla/schema"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jessevdk/go-flags"
	"gopkg.in/tylerb/graceful.v1"
)

// Content-Types
const (
	ApplicationJSON = "application/json"
	TextPlain       = "text/plain; charset=utf-8"
)

// HandlerFunc is the type of handler used by Server
type HandlerFunc func(w http.ResponseWriter, r *http.Request) *Error

// AuthenticatedHandlerFunc is a type of http.handler that requires authorization
type AuthenticatedHandlerFunc func(w http.ResponseWriter, r *http.Request, t schema.Token) *Error

// Config holds options for the server
type Config struct {
	// ConfigPath is the filepath to the javascript server configuration
	ConfigPath string `long:"config-path" description:"path to the javascript config file" default:"./config.js" env:"ARLA_CONFIG_PATH"`
	// Secret is used for signing authentication tokens
	Secret string `long:"secret" description:"secret to use for signing authentication tokens" required:"true" env:"ARLA_SECRET"`
	// DataDir is the filepath to where data will be stored
	DataDir string `long:"data-dir" description:"path to persistant data storage" default:"/var/state" required:"true" env:"ARLA_DATA_DIR"`
	// ListenAddr is the address the HTTP server binds to
	ListenAddr string `long:"listen-addr" description:"address and port to bind http server to" default:":80" required:"true" env:"ARLA_LISTEN_ADDR"`
	// GraceDuration is the time allowed to finishing serving requests during shutdown
	GraceDuration int `long:"grace-duration" description:"time allowed in seconds to finish serving requests during shutdown" default:"1" required:"true" env:"ARLA_GRACE_DURATION"`
	// MaxConnections sets the number of database connections allowed
	MaxConnections int `long:"max-connections" description:"max number of database connections" default:"100" required:"true" env:"ARLA_MAX_CONNECTIONS"`
	// Debug enables debug log messages
	Debug bool `long:"debug" description:"enable verbose debug error logging"`
}

// Server is an HTTP server
type Server struct {
	cfg      Config
	info     *schema.Info
	qs       querystore.Engine
	ms       *mutationstore.Log
	mux      *http.ServeMux
	http     *graceful.Server
	wg       sync.WaitGroup
	stopping bool
}

// Launch the querystore
func (s *Server) startQueryEngine() (err error) {
	if s.qs != nil {
		return nil
	}
	// init query store
	qscfg := &querystore.Config{
		Path:           s.cfg.ConfigPath,
		MaxConnections: s.cfg.MaxConnections,
		LogLevel:       querystore.DEBUG,
	}
	if s.cfg.Debug {
		qscfg.LogLevel = querystore.DEBUG
	}
	s.qs, err = querystore.New(qscfg)
	if err != nil {
		time.Sleep(3 * time.Second) // TODO: exit too soon and you won't see the logs
		return fmt.Errorf("failed to start query engine: %s", err)
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if s.qs != nil {
			if err := s.qs.Wait(); err != nil {
				fmt.Println("postgresql exited: ", err)
			}
		}
		s.qs = nil
		fmt.Println("queryengine shutdown")
	}()
	s.info, err = s.qs.Info()
	if err != nil {
		return err
	}
	fmt.Println("api version", s.info.Version)
	return nil
}

// startLog launches the data store that logs all mutations
func (s *Server) startLog() (err error) {
	if s.ms != nil {
		return nil
	}
	filename := filepath.Join(s.cfg.DataDir, "datastore")
	s.ms, err = mutationstore.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to start mutationstore: %s", err)
	}
	return nil
}

// replayLog sends all previous mutations to the querystore
func (s *Server) replayLog() (err error) {
	start := time.Now()
	defer func() {
		if err == nil {
			fmt.Printf("%d mutations replayed in %s\n", s.ms.Len(), time.Since(start))
		}
	}()
	oldLogLevel := s.qs.GetLogLevel()
	s.qs.SetLogLevel(querystore.ERROR)
	w, err := s.qs.NewWriter()
	if err != nil {
		return fmt.Errorf("failed to create mutation writer: %s", err)
	}
	defer func() {
		if e := w.Close(); e != nil && err == nil {
			err = e
		}
	}()
	if _, err := s.ms.WriteTo(w); err != nil {
		return fmt.Errorf("error streaming mutations to querystore: %s", err)
	}
	s.qs.SetLogLevel(oldLogLevel)
	return nil
}

// login writes an access token to the writer if the user is authenticated
func (s *Server) login(w http.ResponseWriter, vals string) *Error {
	claims, err := s.qs.Authenticate(vals)
	if err != nil {
		return authError(err)
	}
	// create JWT
	token := jwt.New(jwt.SigningMethodHS256)
	for k, v := range claims {
		token.Claims[k] = v
	}
	token.Claims["exp"] = time.Now().Add(time.Hour * 72).Unix()
	accessToken, err := token.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return internalError(err)
	}
	enc := json.NewEncoder(w)
	err = enc.Encode(&struct {
		AccessToken string `json:"access_token,omitempty"`
	}{
		AccessToken: accessToken,
	})
	if err != nil {
		return internalError(err)
	}
	return nil
}

// registrationHandler processes creates a new user account by passing the
// JSON request body to the javascript function defined at arla.cfg.register
// If successful this same request payload is then passed to arla.cfg.authenticate
// to login the user and return an access token.
func (s *Server) registrationHandler(w http.ResponseWriter, r *http.Request) *Error {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return userError(err)
	}
	if s.qs == nil {
		return tempError()
	}
	// ask queryengine to register new user
	m, err := s.qs.Register(string(b))
	if err != nil {
		return userError(err)
	}
	// attempt the mutation
	if s.qs == nil {
		return tempError()
	}
	err = s.qs.Mutate(m)
	if err != nil {
		return userError(err)
	}
	// commit the mutation to the log
	if s.ms == nil {
		return tempError()
	}
	err = s.ms.Write(m)
	if err != nil {
		return internalError(err)
	}
	// login
	return s.login(w, string(b))
}

// infoHandler returns introspection info about the server.
func (s *Server) infoHandler(w http.ResponseWriter, r *http.Request) *Error {
	enc := json.NewEncoder(w)
	if err := enc.Encode(s.info); err != nil {
		return internalError(err)
	}
	return nil
}

// authenticationHandler processes authentication requests by passing the JSON
// request body to the javascript function defined in arla.cfg.authenticate
// if authentication is successful then an access token is returned.
func (s *Server) authenticationHandler(w http.ResponseWriter, r *http.Request) *Error {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return userError(err)
	}
	return s.login(w, string(b))
}

// execHandler reads a Mutation JSON from the request body, executes it
// in the queryengine, writes it to disk in via the mutation log and returns
// a status of whether that was all a success or not.
func (s *Server) execHandler(w http.ResponseWriter, r *http.Request, t schema.Token) *Error {
	// read the mutation json
	var m schema.Mutation
	err := json.NewDecoder(r.Body).Decode(&m)
	if err != nil {
		return userError(err)
	}
	m.Token = t
	// send to query engine
	if s.qs == nil {
		return tempError()
	}
	err = s.qs.Mutate(&m)
	if err != nil {
		return userError(err)
	}
	// write to store
	if s.ms == nil {
		return tempError()
	}
	err = s.ms.Write(&m)
	if err != nil {
		return userError(err)
	}
	// return ok
	err = json.NewEncoder(w).Encode(&struct {
		// ID      schema.UUID `json:"id,omitempty"`
		Success bool `json:"success"`
	}{
		// ID:      m.ID,
		Success: true,
	})
	if err != nil {
		return internalError(err)
	}
	return nil
}

// queryHandler accepts a GraphQL-like query in the request body and executes
// it against the data in the query engine. The response is JSON.
func (s *Server) queryHandler(w http.ResponseWriter, r *http.Request, t schema.Token) *Error {
	q := &schema.Query{
		Token: t,
	}
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		return userError(err)
	}
	if s.qs == nil {
		return tempError()
	}
	if err := s.qs.Query(q, w); err != nil {
		return userError(err)
	}
	return nil
}

// enableCORS sets headers to allow CORS.
// XXX: by default we allow CORS requests ... but there should be a way to configure it.
func (s *Server) enableCORS(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	}
}

// addHandler attaches a HandleFunc to the http server.
func (s *Server) addHandler(path string, fn HandlerFunc) {
	s.mux.HandleFunc(path, s.wrapHandler(fn))
}

// addAuthenticatedHandler attaches a AuthenticatedHandleFunc to the http server
func (s *Server) addAuthenticatedHandler(path string, fn AuthenticatedHandlerFunc) {
	s.addHandler(path, s.wrapAuthenticatedHandler(fn))
}

// wrapHandler converts our HandlerFunc into an http.HandlerFunc.
// It ensures that the error responses are always JSON encoded
func (s *Server) wrapHandler(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// set default response type
		w.Header().Set("Content-Type", ApplicationJSON)
		// enable CORS
		s.enableCORS(w, r)
		if r.Method == "OPTIONS" {
			return
		}
		// call handler
		if err := fn(w, r); err != nil {
			// handle errors
			if s.cfg.Debug {
				fmt.Fprintf(os.Stderr, "DEBUG: %v", err)
			}
			w.WriteHeader(err.code)
			enc := json.NewEncoder(w)
			if fatal := enc.Encode(err); fatal != nil {
				fmt.Fprintf(os.Stderr, "error during error handling: %v", fatal)
				return
			}
		}
	}
}

// wrapAuthenticatedHandler converts an AuthenticatedHandleFunc to a HandlerFunc
func (s *Server) wrapAuthenticatedHandler(fn AuthenticatedHandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) *Error {
		// get token from request
		token, err := jwt.ParseFromRequest(r, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(s.cfg.Secret), nil
		})
		if err != nil {
			return authError(err)
		}
		if !token.Valid {
			if ve, ok := err.(*jwt.ValidationError); ok {
				if ve.Errors&jwt.ValidationErrorMalformed != 0 {
					return authError(fmt.Errorf("malformed token"))
				}
				if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
					return authError(fmt.Errorf("token expired"))
				}
			}
			return authError(fmt.Errorf("invalid token"))
		}
		t := schema.Token(token.Claims)
		return fn(w, r, t)
	}
}

// startHTTP launches the http server
func (s *Server) startHTTP() error {
	if s.http != nil {
		return nil
	}
	shutdownExpected := false
	s.http = &graceful.Server{
		Timeout: time.Duration(s.cfg.GraceDuration) * time.Second,
		Server: &http.Server{
			Addr:    s.cfg.ListenAddr,
			Handler: s.mux,
		},
		ShutdownInitiated: func() {
			fmt.Println("http server shutting down")
			shutdownExpected = true
		},
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		fmt.Println("http server started")
		if err := s.http.ListenAndServe(); err != nil {
			if !shutdownExpected {
				fmt.Println("ListenAndServe: ", err)
			}
		}
		s.http = nil
		fmt.Println("http server shutdown")
	}()
	return nil
}

// Start launches the Server
func (s *Server) Start() (err error) {
	defer func() {
		if err != nil {
			s.Stop()
		}
	}()
	if err = s.startQueryEngine(); err != nil {
		return
	}
	if err = s.startLog(); err != nil {
		return
	}
	if err = s.replayLog(); err != nil {
		fmt.Println("FAILED TO REPLAY MUTATIONS", err)
		return
	}
	if err = s.startHTTP(); err != nil {
		return
	}
	return nil
}

// Wait blocks while the server is running
func (s *Server) Wait() error {
	s.wg.Wait()
	return nil
}

// Run is same as Start followed by Wait
func (s *Server) Run() error {
	if err := s.Start(); err != nil {
		return err
	}
	return s.Wait()
}

// Stop shutsdown the server and blocks until complete
func (s *Server) Stop() error {
	if s.stopping {
		return nil
	}
	s.stopping = true
	defer func() {
		s.stopping = false
	}()
	var errs []string
	if s.http != nil {
		s.http.Stop(1 * time.Second)
		s.http = nil
	}
	if s.qs != nil {
		if err := s.qs.Stop(); err != nil {
			errs = append(errs, err.Error())
		}
		s.qs = nil
	}
	if s.ms != nil {
		if err := s.ms.Close(); err != nil {
			errs = append(errs, err.Error())
		}
		s.ms = nil
	}
	if err := s.Wait(); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to shutdown cleanly: %s", strings.Join(errs, " AND "))
	}
	return nil
}

// New creates a new server with all required fields set
func New(cfg Config) *Server {
	s := &Server{
		cfg: cfg,
		mux: http.NewServeMux(),
	}
	s.addHandler("/info", s.infoHandler)
	s.addHandler("/register", s.registrationHandler)
	s.addHandler("/authenticate", s.authenticationHandler)
	s.addAuthenticatedHandler("/exec", s.execHandler)
	s.addAuthenticatedHandler("/query", s.queryHandler)
	s.mux.Handle("/", http.FileServer(http.Dir("/app/public")))
	return s
}

func main() {
	var cfg Config
	if _, err := flags.ParseArgs(&cfg, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := New(cfg).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
