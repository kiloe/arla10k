package querystore

import (
	"arla/schema"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)
import "github.com/jackc/pgx"

func init() {
	pgx.DefaultTypeFormats["json"] = pgx.BinaryFormatCode
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
	cmd  *exec.Cmd
	quit chan (error)
}

// Stop disconnects and shutsdown the queryengine
func (p *postgres) Stop() error {
	return p.cmd.Process.Kill()
}

// Mutate applies a schema.Mutation to the data
func (p *postgres) Mutate(m *schema.Mutation) error {
	if !m.UserID.Valid() {
		return fmt.Errorf("cannot process mutation without user id")
	}
	p.execMu.Lock()
	defer p.execMu.Unlock()
	args, err := json.Marshal(m.Args)
	if err != nil {
		return err
	}
	_, err = p.execConn.Exec("select arla_exec($1::uuid, $2::text, $3::json)", m.UserID, m.Name, string(args))
	return err
}

// Query executes an Arla query and writes the JSON response into w
func (p *postgres) Query(uid schema.UUID, query string, w io.Writer) error {
	out := jsonbytes{w: w}
	r := p.queryPool.QueryRow("select arla_query($1::uuid, $2::text)", uid, query)
	if err := r.Scan(&out); err != nil {
		return err
	}
	return nil
}

// Start spawns a postgres instance, configures it using the
// supplied actions.js and schema.js paths and creates a connection pool.
func (p *postgres) Start() (err error) {
	p.quit = make(chan error)
	p.pgcfg, err = pgx.ParseEnvLibpq()
	if err != nil {
		return
	}
	p.pgcfg.User = "postgres"
	p.pgcfg.Database = "arla"
	p.pgcfg.Host = "/var/run/postgresql/"
	if err = p.spawn(); err != nil {
		return
	}
	p.execConn, err = pgx.Connect(p.pgcfg)
	if err != nil {
		return
	}
	p.queryPool, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:     p.pgcfg,
		MaxConnections: 5,
	})
	if err != nil {
		return
	}
	return nil
}

func (p *postgres) Wait() error {
	return <-p.quit
}

func (p *postgres) spawn() (err error) {
	p.cmd, err = p.command("pg_ctlcluster", "--foreground", "9.4", "main", "start")
	if err != nil {
		return err
	}
	if err := p.cmd.Start(); err != nil {
		return err
	}
	go func() {
		p.quit <- p.cmd.Wait()
	}()
	// wait until responsive
	select {
	case <-p.pollForReady():
		break
	case <-time.After(10 * time.Second):
		p.Stop()
		return errors.New("timeout waiting for postgres to start accepting connections")
	}
	// init
	return p.init()
}

// command is exec.Command but preconfigured for postgres user and always
// looks up first argument using LookPath
func (p *postgres) command(name string, args ...string) (*exec.Cmd, error) {
	pgUser, err := user.Lookup("postgres")
	if err != nil {
		return nil, err
	}
	uid, err := strconv.ParseUint(pgUser.Uid, 10, 32)
	if err != nil {
		return nil, err
	}
	gid, err := strconv.ParseUint(pgUser.Gid, 10, 32)
	if err != nil {
		return nil, err
	}
	exe, err := exec.LookPath(name)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = append(os.Environ(), []string{
		"PGUSER=postgres",
		"PGDATABASE=arla",
	}...)
	return cmd, nil
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
	ch := make(chan (bool))
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			if err := p.run("pg_isready"); err != nil {
				log.Println(err)
				continue
			}
			ch <- true
			break
		}
	}()
	return ch
}

// initialize the database
func (p *postgres) init() error {
	p.run("dropdb", "arla")
	if err := p.run("createdb"); err != nil {
		return err
	}
	// compile js
	cmd, err := p.command("browserify",
		p.cfg.Path, "-t", "[",
		"/usr/local/lib/node_modules/babelify",
		"--modules", "common",
		"]")
	if err != nil {
		return err
	}
	var js bytes.Buffer
	cmd.Stdout = &js
	if err := cmd.Run(); err != nil {
		return err
	}
	// compile sql
	sql := strings.Replace(postgresInitScript, "__INDEX_JS__", string(js.Bytes()), 1)
	sql = strings.Replace(sql, "__RUNTIME__", string(postgresRuntimeScript), 1)
	// exec sql
	cmd, err = p.command("psql", "-v", "ON_ERROR_STOP=1")
	if err != nil {
		return err
	}
	cmd.Stdin = strings.NewReader(sql)
	err = cmd.Run()
	if err != nil {
		// extract line no of error
		// dump context of sql to help debugging
		cmd, _ = p.command("cat", "-n", "-")
		cmd.Stdin = strings.NewReader(sql)
		cmd.Run()

		return err
	}
	return nil
}
