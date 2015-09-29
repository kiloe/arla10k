package querystore

import (
	"arla/schema"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)
import "github.com/jackc/pgx"

var devNull *os.File

var postgresInitUserOffset int

func init() {
	pgx.DefaultTypeFormats["json"] = pgx.BinaryFormatCode
	// blackhole
	devNull, _ = os.OpenFile(os.DevNull, os.O_WRONLY, 0644)
}

// jsonbytes is directly writes the response of a ::json type to it's io.Writer
type jsonbytes struct {
	w io.Writer
}

// Scan implements pgx.Scanner
func (jb *jsonbytes) Scan(vr *pgx.ValueReader) error {
	if vr.Type().DataTypeName != "json" {
		return pgx.SerializationError(fmt.Sprintf("jsonstream.Scan cannot decode %s (OID %d)", vr.Type().DataTypeName, vr.Type().DataType))
	}
	// ensure binary
	switch vr.Type().FormatCode {
	case pgx.TextFormatCode:
		return errors.New("jsonstream text format not implemented")
	case pgx.BinaryFormatCode:
		chunk := int32(8192)
		for {
			n := vr.Len()
			if n <= 0 {
				break
			}
			if n < chunk {
				chunk = n
			}
			b := vr.ReadBytes(chunk)
			sent, err := jb.w.Write(b)
			if err != nil {
				return err
			}
			if sent != len(b) {
				return errors.New("number of bytes written does not match")
			}
		}
	default:
		return fmt.Errorf("jsonstream: unknown format %v", vr.Type().FormatCode)
	}
	return vr.Err()
}

// postgres implements querystore.Engine using postgresql
type postgres struct {
	// execConn is used to serialize mutations to the data
	execConn *pgx.Conn
	// execMu is a mutex for executing mutations
	execMu sync.Mutex
	// queryPool is used for reads
	queryPool *pgx.ConnPool
	// options
	cfg   *Config
	pgcfg pgx.ConnConfig
	// block/unblock Start
	cmd   *exec.Cmd
	quit  chan (error)
	ready chan (bool)
	// streaming sql
	writeCmd *exec.Cmd
	// log output
	log *LogFormatter
	// user cfg
	info *schema.Info
	// max number of db connections
	maxConnections int
}

func (p *postgres) SetLogLevel(level logLevel) {
	p.log.Level = level
}

func (p *postgres) GetLogLevel() logLevel {
	return p.log.Level
}

// Stop disconnects and shutsdown the queryengine
func (p *postgres) Stop() (err error) {
	if p.cmd == nil {
		return nil
	}
	if p.cmd.Process == nil {
		return nil
	}
	p.cmd.Process.Signal(os.Kill)
	return nil
}

// Info returns details used for introspection
func (p *postgres) Info() (*schema.Info, error) {
	return p.info, nil
}

// Mutate applies a schema.Mutation to the data
func (p *postgres) Mutate(m *schema.Mutation) error {
	if m.Name == "" {
		return fmt.Errorf("invalid mutation name")
	}
	p.execMu.Lock()
	defer p.execMu.Unlock()
	m.Version = p.info.Version
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = p.execConn.Exec("select arla_exec($1::json)", string(b))
	return err
}

// Query executes an Arla query and writes the JSON response into w
func (p *postgres) Query(q *schema.Query, w io.Writer) error {
	out := jsonbytes{w: w}
	b, err := json.Marshal(q)
	if err != nil {
		return err
	}
	r := p.queryPool.QueryRow("select arla_query($1::json)", string(b))
	if err := r.Scan(&out); err != nil {
		return err
	}
	return nil
}

// Authenticate returns the token claims for the given json values
func (p *postgres) Authenticate(vals string) (schema.Token, error) {
	var s string
	r := p.queryPool.QueryRow("select arla_authenticate($1::json)", vals)
	if err := r.Scan(&s); err != nil {
		return nil, err
	}
	t := schema.Token{}
	if err := json.Unmarshal([]byte(s), &t); err != nil {
		return nil, err
	}
	return t, nil
}

// Register returns a mutation that will be used to create a user
func (p *postgres) Register(vals string) (*schema.Mutation, error) {
	var s string
	r := p.queryPool.QueryRow("select arla_register($1::json)", vals)
	if err := r.Scan(&s); err != nil {
		return nil, err
	}
	var m schema.Mutation
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Copy the config files into the data dir
func (p *postgres) cpConfig(name string) (err error) {
	dataDir := os.Getenv("PGDATA")
	src := filepath.Join(dataDir, "..", name)
	fmt.Println("COPY", src, dataDir)
	err = p.run("cp", "-f", src, dataDir)
	if err != nil {
		return fmt.Errorf("failed to install %s: %v", name, err)
	}
	return nil
}

func (p *postgres) initdb() (err error) {
	err = p.run("initdb", "--nosync", "--noclean")
	if err != nil {
		return err
	}
	err = p.cpConfig("postgresql.conf")
	if err != nil {
		return err
	}
	err = p.cpConfig("pg_hba.conf")
	if err != nil {
		return err
	}
	return nil
}

// Start spawns a postgres instance, configures it using the
// supplied actions.js and schema.js paths and creates a connection pool.
func (p *postgres) Start() (err error) {
	p.quit = make(chan error, 1)
	p.pgcfg, err = pgx.ParseEnvLibpq()
	if err != nil {
		return
	}
	p.pgcfg.User = "postgres"
	p.pgcfg.Database = "arla"
	p.pgcfg.Host = "/var/run/postgresql/"
	if err = p.initdb(); err != nil {
		return
	}
	if err = p.spawn(); err != nil {
		return
	}
	p.execConn, err = pgx.Connect(p.pgcfg)
	if err != nil {
		return
	}
	p.queryPool, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:     p.pgcfg,
		MaxConnections: p.maxConnections,
	})
	if err != nil {
		return
	}
	// load app info
	var s string
	r := p.queryPool.QueryRow("select arla_info()")
	if err := r.Scan(&s); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(s), &p.info); err != nil {
		return err
	}
	return nil
}

func (p *postgres) NewWriter() (w io.WriteCloser, err error) {
	return newPgWriter(p)
}

func (p *postgres) Wait() error {
	err := <-p.quit
	return err
}

func (p *postgres) spawn() (err error) {
	p.cmd, err = p.command(
		"postgres",
		"-k", "/var/run/postgresql",
		"-c", fmt.Sprintf("max_connections=%d", p.maxConnections+1),
	)
	if err != nil {
		return err
	}
	p.cmd.Stderr = p.log
	p.cmd.Stdout = p.log
	if err := p.cmd.Start(); err != nil {
		return err
	}
	go func() {
		_, err := p.cmd.Process.Wait()
		p.quit <- err
		close(p.quit)
	}()
	// wait until responsive
	select {
	case <-p.pollForReady():
		break
	case <-time.After(10 * time.Second):
		p.Stop()
		return errors.New("timeout waiting for postgres to start accepting connections")
	case err, ok := <-p.quit:
		if ok && err != nil {
			return fmt.Errorf("failed to start postgres: %v", err)
		}
		return errors.New("postgres progress exited during startup")
	}
	// init
	return p.init()
}

// command is exec.Command but preconfigured for postgres user and always
// looks up first argument using LookPath
func (p *postgres) command(name string, args ...string) (cmd *exec.Cmd, err error) {
	uid, err := getUid("postgres")
	if err != nil {
		return nil, err
	}
	gid, err := getGid("postgres")
	if err != nil {
		return nil, err
	}
	exe, err := exec.LookPath(name)
	if err != nil {
		return nil, err
	}
	cmd = exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid}
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.Env = append(os.Environ(), []string{
		"PGUSER=postgres",
		"PGDATABASE=arla",
		"PGHOST=/var/run/postgresql",
	}...)
	return
}

// run is a shortcut for p.command + start + wait
func (p *postgres) run(exe string, args ...string) error {
	cmd, err := p.command(exe, args...)
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

// forForReady returns a channel that signals when daemon is up
func (p *postgres) pollForReady() <-chan (bool) {
	fmt.Printf("starting postgres..")
	ch := make(chan (bool))
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			if err := p.run("pg_isready", "-d", "postgres"); err != nil {
				fmt.Printf(".")
				continue
			}
			ch <- true
			fmt.Printf("...ready\n")
			break
		}
	}()
	return ch
}

// initialize the database
func (p *postgres) init() error {
	if err := p.run("createdb"); err != nil {
		p.run("dropdb", "arla")
		if err := p.run("createdb"); err != nil {
			return err
		}
	}
	// compile js
	cmd, err := p.command("browserify",
		p.cfg.Path, "-t", "[",
		"/usr/lib/node_modules/babelify",
		"--modules", "common",
		"--stage", "0",
		"]")
	if err != nil {
		return err
	}
	var js bytes.Buffer
	cmd.Stdout = &js
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to compile app source: %s", err)
	}
	marker := "//CONFIG//"
	// record the offset to usercode
	postgresInitUserOffset = strings.Index(postgresInitScript, marker)
	// compile sql
	sql := strings.Replace(postgresInitScript, marker, string(js.Bytes()), 1)
	p.log.src = &sql
	// exec sql
	cmd, err = p.command("psql", "-v", "ON_ERROR_STOP=1")
	if err != nil {
		return err
	}
	cmd.Stdin = strings.NewReader(sql)
	cmd.Stderr = p.log // wire up client output to server logs
	cmd.Stdout = p.log // wire up client output to server logs
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to initialize arla: %s", err)
	}
	return nil
}
