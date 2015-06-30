package querystore

import (
	"arla/schema"
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"code.google.com/p/go-uuid/uuid"
)
import "github.com/jackc/pgx"

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
	quit chan (error)
}

// Stop disconnects and shutsdown the queryengine
func (p *postgres) Stop() error {
	if p.quit != nil {
		p.quit <- nil
		p.quit = nil
	}
	return nil
}

// Mutate applies a schema.Mutation to the data
func (p *postgres) Mutate(m *schema.Mutation) error {
	p.execMu.Lock()
	defer p.execMu.Unlock()
	_, err := p.execConn.Exec("select arla_exec($1::text, $2::json)", m.Name, m.Args)
	return err
}

// Query executes an Arla query and returns the response as JSON
// encoded bytes.
func (p *postgres) Query(uid uuid.UUID, query string) (json []byte, err error) {
	p.queryPool.Query("select arla_query($1::uuid, $2::text)", uid, query)
	return
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
	return <-p.quit
}

func (p *postgres) spawn() error {
	cmd, err := p.command("pg_ctlcluster", "--foreground", "9.4", "main", "start")
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		cmd.Wait()
		log.Println("onExit: postgres daemon exited")
		p.Stop()
	}()
	// wait until responsive
	select {
	case <-p.pollForReady():
		break
	case <-time.After(30 * time.Second):
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
