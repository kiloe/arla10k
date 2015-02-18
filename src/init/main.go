// init is arla's process supervisor and start-up script.
//
// It's responsible for:
// * starting postgres
// * migrating the postgres schema (from schema.js)
// * waiting for postgres to start
// * launching arla's http server
// * forwarding on signals to chlid processes
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"
	"time"
)

const (
	postgresCmd        = "/usr/bin/pg_ctlcluster"
	postgresReadyCmd   = "/usr/bin/pg_isready"
	postgresData       = "/var/lib/postgresql/9.4/main"
	postgresConf       = "/etc/postgresql/9.4/main/postgresql.conf"
	postgresInitScript = "/var/lib/arla/bin/initdb"
	arlaServerCmd      = "/usr/local/bin/arla_server"
	arlaTestCmd        = "/usr/local/bin/arla_tests"
	libdir             = "/var/lib/arla/"
)

var env = append(os.Environ(), []string{
	"PGUSER=postgres",
	"PGDATABASE=arla",
}...)

var (
	signals = make(chan os.Signal, 5)
)

func pgIsReady() bool {
	cmd := exec.Command(postgresReadyCmd)
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		return false
	}
	return cmd.Wait() == nil
}

// startPostgres launches
func startPostgres() (onExit chan error, err error) {
	pgUser, err := user.Lookup("postgres")
	if err != nil {
		return onExit, err
	}
	uid, err := strconv.ParseUint(pgUser.Uid, 10, 32)
	if err != nil {
		return onExit, err
	}
	gid, err := strconv.ParseUint(pgUser.Gid, 10, 32)
	if err != nil {
		return onExit, err
	}
	onExit = make(chan error)
	cmd := exec.Command(postgresCmd, "--foreground", "9.4", "main", "start")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Start(); err != nil {
		return onExit, err
	}
	go func() {
		onExit <- cmd.Wait()
	}()
	// wait until responsive
	timeout := time.After(30 * time.Second)
	for !pgIsReady() {
		select {
		case <-time.After(500 * time.Millisecond):
			continue
		case <-signals:
			return onExit, errors.New("signal received while waiting for postgres to start")
		case <-timeout:
			return onExit, errors.New("timeout waiting for postgres to start accepting connections")
		}
	}
	// run the initialisation script
	init := exec.Command(postgresInitScript)
	init.Stderr = os.Stderr
	init.Stdout = os.Stdout
	init.Env = env
	init.Dir = libdir
	if err := init.Run(); err != nil {
		return onExit, err
	}
	return onExit, nil
}

func startApiServer() (onExit chan error, err error) {
	onExit = make(chan error)
	cmd := exec.Command(arlaServerCmd)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = libdir
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		return onExit, err
	}
	go func() {
		onExit <- cmd.Wait()
	}()
	return onExit, nil
}

func runTests() (err error) {
	cmd := exec.Command(arlaTestCmd)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	//cmd.Dir = filepath.Join(libdir, "api")
	cmd.Env = env
	return cmd.Run()
}

// start launches init
func start() error {
	// Listen for signals send to this process so they can be forwarded on
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Boot postgres
	pgExit, err := startPostgres()
	if err != nil {
		return err
	}

	if true {
		return runTests()
	}
	// Boot API server
	apiExit, err := startApiServer()
	if err != nil {
		return err
	}

	for {
		select {
		case err := <-pgExit:
			return err
		case err := <-apiExit:
			return err
		case sig := <-signals:
			return fmt.Errorf("received unexpected signal: %v", sig)
		}
	}
}

func main() {
	if err := start(); err != nil {
		log.Fatal(err)
	}
}
