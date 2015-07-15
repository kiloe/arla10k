package querystore

import (
	"io"
	"os/exec"
)

type pgWriter struct {
	p   *postgres
	cmd *exec.Cmd
	err chan error
	io.WriteCloser
}

func newPgWriter(p *postgres) (pgw *pgWriter, err error) {
	pgw = &pgWriter{
		p: p,
	}

	if pgw.cmd, err = p.command("psql", "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1"); err != nil {
		return
	}
	if pgw.WriteCloser, err = pgw.cmd.StdinPipe(); err != nil {
		return
	}
	pgw.cmd.Stdout = devNull
	pgw.cmd.Stderr = devNull
	if err = pgw.cmd.Start(); err != nil {
		return
	}

	pgw.err = make(chan error)
	go func() {
		pgw.err <- pgw.cmd.Wait()
	}()
	return
}

func (pgw *pgWriter) Close() error {
	pgw.WriteCloser.Close()
	return <-pgw.err
}
