package main

import (
	"arla/mutationstore"
	"arla/querystore"
	"arla/schema"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var secret = []byte("mysecretkey")

type opts struct {
	ConfigPath string
}

// ErrorResponse is the format of any errors from the API
type ErrorResponse struct {
	Code  int    `json:"code,omitempty"`
	Error string `json:"error"`
}

// AuthResponse is the format returned by successful calls to /authenticate
type AuthResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// ExecResponse is the format returned by calls to /exec
type ExecResponse struct {
	ID      schema.UUID `json:"id"`
	Success bool        `json:"success"`
}

func authenticate(r *http.Request) (schema.Token, error) {
	token, err := jwt.ParseFromRequest(r, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, fmt.Errorf("malformed token")
			} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
				return nil, fmt.Errorf("token expired")
			} else {
				return nil, fmt.Errorf("invalid token")
			}
		} else {
			return nil, fmt.Errorf("invalid token")
		}
	}
	t := make(schema.Token)
	for k, v := range token.Claims {
		if s, ok := v.(string); ok {
			t[k] = s
		}
	}
	return t, nil
}

func login(w http.ResponseWriter, qs querystore.Engine, vals string) {
	claims, err := qs.Authenticate(vals)
	if err != nil {
		fail(w, err, http.StatusUnauthorized)
		return
	}
	// create JWT
	token := jwt.New(jwt.SigningMethodHS256)
	for k, v := range claims {
		token.Claims[k] = v
	}
	token.Claims["exp"] = time.Now().Add(time.Hour * 72).Unix()
	accessToken, err := token.SignedString(secret)
	if err != nil {
		fail(w, err, http.StatusInternalServerError)
		return
	}
	enc := json.NewEncoder(w)
	err = enc.Encode(&AuthResponse{
		AccessToken: accessToken,
	})
	if err != nil {
		fail(w, err, http.StatusInternalServerError)
		return
	}
}

// fail is like http.Error() but always returns JSON
func fail(w http.ResponseWriter, err error, status int) {
	e := &ErrorResponse{
		Error: err.Error(),
	}
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	if err := enc.Encode(e); err != nil {
		log.Println("error during error handling: ", err.Error())
		http.Error(w, "fatal error during error handling", http.StatusInternalServerError)
		return
	}
}

func main() {
	// init query store
	fmt.Println("starting query service...")
	qs, err := querystore.New(&querystore.Config{
		Path:     "/app/index.js",
		LogLevel: querystore.DEBUG,
	})
	if err != nil {
		time.Sleep(5 * time.Second) // TODO: exit too soon and you won't see the logs
		log.Println(err)
		return
	}
	go func() {
		if err := qs.Wait(); err != nil {
			log.Fatal("qs died! ", err)
		}
	}()
	// extract app config
	// cfg := qs.GetConfig()

	// init action store
	fmt.Println("starting mutation service...")
	ms, err := mutationstore.Open("/var/state/datastore")
	if err != nil {
		log.Fatal("ms open", err)
	}
	// silence logs for a bit and replay
	fmt.Println("replaying mutations...")
	start := time.Now()
	oldLogLevel := qs.GetLogLevel()
	qs.SetLogLevel(querystore.ERROR)
	w, err := qs.NewWriter()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := ms.WriteTo(w); err != nil {
		log.Fatal(err)
	}
	w.Close()
	qs.SetLogLevel(oldLogLevel)
	fmt.Printf("%d mutations replayed in %s\n", ms.Len(), time.Since(start))

	// handler to register new users
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
		// ask queryengine for transformation
		m, err := qs.Register(string(b))
		if err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
		// attempt the mutation
		err = qs.Mutate(m)
		if err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
		// commit the mutation to the log
		err = ms.Write(m)
		if err != nil {
			fail(w, err, http.StatusInternalServerError)
			return
		}
		// login
		login(w, qs, string(b))
	})

	// add handler to authenticate existing users
	http.HandleFunc("/authenticate", func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
		// ask queryengine for auth
		login(w, qs, string(b))
	})

	// add handler for mutations/actions
	http.HandleFunc("/exec", func(w http.ResponseWriter, r *http.Request) {
		// read the mutation json
		dec := json.NewDecoder(r.Body)
		var m schema.Mutation
		err := dec.Decode(&m)
		if err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
		// check the token
		m.Token, err = authenticate(r)
		if err != nil {
			fail(w, err, http.StatusUnauthorized)
			return
		}
		// send to query engine
		err = qs.Mutate(&m)
		if err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
		// write to store
		err = ms.Write(&m)
		if err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
		// return ok
		enc := json.NewEncoder(w)
		err = enc.Encode(&ExecResponse{
			ID:      m.ID,
			Success: true,
		})
		if err != nil {
			fail(w, err, http.StatusInternalServerError)
		}
	})

	// add handler for query
	http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
		// check the token
		t, err := authenticate(r)
		if err != nil {
			fail(w, err, http.StatusForbidden)
			return
		}
		err = qs.Query(t, string(b), w)
		if err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
	})

	// start http server
	log.Println("arla is ready")
	fs := http.FileServer(http.Dir("/app/public"))
	http.Handle("/", fs)

	http.ListenAndServe(":80", nil)
}
