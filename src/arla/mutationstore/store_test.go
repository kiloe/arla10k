package mutationstore

import (
	"arla/schema"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"code.google.com/p/go-uuid/uuid"
)

var tmpdir string

func TestMain(m *testing.M) {
	var err error
	tmpdir, err = ioutil.TempDir("", "waltest")
	if err != nil {
		log.Fatal(err)
	}
	status := m.Run()
	if err = os.RemoveAll(tmpdir); err != nil {
		log.Fatal(err)
	}
	os.Exit(status)
}

func TestWriteThenRead(t *testing.T) {
	filename := filepath.Join(tmpdir, "TestWriteThenRead")
	in := &schema.Mutation{
		Name: "exampleOp",
	}
	// write 10 recrods
	log, err := Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.Write(in); err != nil {
		t.Fatal(err)
	}
	n := 0
	for m := range log.Replay() {
		n++
		if in.Name != m.Name {
			t.Fatalf("expected in.Name (%v) to match out.Name (%v)", in.Name, m.Name)
		}
	}
	if n != 1 {
		t.Fatalf("expected exactly 1 record got %d", n)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkWrites(b *testing.B) {
	m := &schema.Mutation{
		ID:   uuid.NewUUID(),
		Name: "exampleOp",
	}
	// remove file if exists
	filename := filepath.Join(tmpdir, "BenchmarkWrites")
	os.Remove(filename)
	// open
	log, err := Open(filename)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	// write alot
	for i := 0; i < b.N; i++ {
		if err := log.Write(m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReads(b *testing.B) {
	m := &schema.Mutation{
		ID:   uuid.NewUUID(),
		Name: "exampleOp",
	}
	// remove file if exists
	filename := filepath.Join(tmpdir, "BenchmarkReads")
	os.Remove(filename)
	// open
	log, err := Open(filename)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		if err := log.Write(m); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	// read
	for i := 0; i < b.N; i++ {
		for _ = range log.Replay() {
			// noop
		}
	}
}
