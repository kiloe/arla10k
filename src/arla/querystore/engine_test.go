package querystore

import (
	"arla/schema"
	"bytes"
	"fmt"
	"log"
	"os"
	"testing"
)

var qs Engine

func TestMain(m *testing.M) {
	fmt.Println("creating engine...")
	qs = New(&Config{
		Path: "/app/test-app/index.js",
	})
	fmt.Println("starting engine...")
	err := qs.Start()
	if err != nil {
		log.Fatal("qs start", err)
	}
	fmt.Println("running tests...")
	st := m.Run()
	qs.Stop()
	os.Exit(st)
}

func TestMutate(t *testing.T) {
	m := &schema.Mutation{
		ID:     schema.TimeUUID(),
		UserID: schema.TimeUUID(),
		Name:   "exampleOp",
		Args:   []interface{}{1, 2, 3},
	}
	err := qs.Mutate(m)
	if err != nil {
		t.Fatal(err)
	}
}

func TestQuery(t *testing.T) {
	id := schema.TimeUUID()
	var buf bytes.Buffer
	err := qs.Query(id, `
    oneToTen() {
		
		}
  `, &buf)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("buf-->", buf.String())

}
