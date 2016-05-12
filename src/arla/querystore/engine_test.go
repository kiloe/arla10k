package querystore

import (
	"arla/schema"
	"bytes"
	"encoding/json"
	"log"
	"os"
	"testing"
)

var qs Engine

func TestMain(m *testing.M) {
	var err error
	qs, err = New(&Config{
		Path:     "/app/index.js",
		LogLevel: INFO,
	})
	if err != nil {
		log.Fatal(err)
	}
	st := m.Run()
	qs.Stop()
	os.Exit(st)
}

func TestMutate(t *testing.T) {
	m := &schema.Mutation{
		ID:   schema.TimeUUID(),
		Name: "exampleOp",
		Args: []interface{}{1, 2, 3},
	}
	err := qs.Mutate(m)
	if err != nil {
		t.Fatal(err)
	}
}

func TestQuery(t *testing.T) {
	var buf bytes.Buffer
	err := qs.Query(nil, `
    oneToTen() {

		}
  `, &buf)
	if err != nil {
		t.Fatal(err)
	}
	res := struct {
		OneToTen struct {
			Numbers []int
		}
	}{}

	dec := json.NewDecoder(&buf)
	if err := dec.Decode(&res); err != nil {
		t.Fatal(err)
	}
	if len(res.OneToTen.Numbers) != 10 {
		t.Fatal("expected oneToTen() to return an object with property numbers containing an array of 10 numbers")
	}
}
