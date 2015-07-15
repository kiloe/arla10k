// Package mutationstore implements a channel based interface for reading/writing
// a sequential log of Mutations to disk in a safe way.
package mutationstore

import (
	"arla/schema"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Log gives safe sequential access to the log of Mutations
type Log struct {
	filename string
	in       chan (*writeRequest)
	closed   bool
	count    int64
	io.Reader
}

type writeRequest struct {
	m   *schema.Mutation
	err chan (error)
}

// Write a mutation to the Log.
func (l *Log) Write(m *schema.Mutation) error {
	r := &writeRequest{
		m:   m,
		err: make(chan (error)),
	}
	if l.closed {
		return fmt.Errorf("cannot write to closed log")
	}
	l.in <- r
	l.count++
	return <-r.err
}

// Close the log
func (l *Log) Close() error {
	if l.closed {
		return nil
	}
	l.closed = true
	close(l.in)
	return nil
}

// writer is a goroutine that reads from the "in" chan
// and writes the value to disk
func (l *Log) writer() {
	// Open as O_RDWR (which should get lock) and O_DIRECT.
	f, err := os.OpenFile(l.filename, os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for {
		r, ok := <-l.in
		if !ok {
			return
		}
		if r.m == nil {
			r.err <- fmt.Errorf("cannot write nil to wal")
			return
		}
		// serialize mutation and write to disk
		if err := enc.Encode(r.m); err != nil {
			r.err <- fmt.Errorf("wal encoding: %s", err.Error())
			return
		}
		// sync
		if err := f.Sync(); err != nil {
			r.err <- fmt.Errorf("wal sync: %s", err.Error())
			return
		}
		r.err <- nil
		// send to reader
		if l.closed {
			return
		}
	}
}

// Replay returns a channel that emits each item from the log.
func (l *Log) Replay() <-chan (*schema.Mutation) {
	ch := make(chan (*schema.Mutation), 1000)
	go func() {
		f, err := os.OpenFile(l.filename, os.O_RDONLY, 0660)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		dec := json.NewDecoder(f)
		for {
			var m schema.Mutation
			if err := dec.Decode(&m); err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			}
			ch <- &m
		}
		close(ch)
	}()
	return ch
}

// WriteTo writes the log as SQL statements to w
func (l *Log) WriteTo(w io.Writer) (n int64, err error) {
	f, err := os.OpenFile(l.filename, os.O_RDONLY, 0660)
	if err != nil {
		return
	}
	defer f.Close()
	prefix := "select arla_replay('"
	suffix := "'::json);\n"
	s := bufio.NewScanner(f)
	nx := 0
	for s.Scan() {
		b := s.Bytes()
		// skip blank lines
		if len(b) == 0 {
			continue
		}
		// write prefix
		if nx, err = w.Write([]byte(prefix)); err != nil {
			return
		}
		n += int64(nx)
		// write line
		if nx, err = w.Write(b); err != nil {
			return
		}
		n += int64(nx)
		// write suffix
		if nx, err = w.Write([]byte(suffix)); err != nil {
			return
		}
		n += int64(nx)
		l.count++
	}
	if err = s.Err(); err != nil {
		return
	}
	return
}

// Len returns the current number of mutations logged
func (l *Log) Len() int64 {
	return l.count
}

// Open sets up access to a Log for a given filename.
// If filename does not exist, it will be created.
// If a replay channel is returned, then each mutation read from disk
// will be sent to the chan. The chan will be closed once all mutations are read
// Writes will be buffered until all mutations are sent to the chan - so you MUST
// drain this chan before issusing writes!
func Open(filename string) (l *Log, err error) {
	l = &Log{
		filename: filename,
		in:       make(chan (*writeRequest), 1000),
	}
	f, err := os.OpenFile(l.filename, os.O_RDONLY|os.O_CREATE, 0660)
	if err != nil {
		return l, err
	}
	defer f.Close()
	go l.writer()
	return l, nil
}
