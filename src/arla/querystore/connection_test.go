package querystore

import (
	"arla/schema"
	"log"
	"os"
	"testing"

	"code.google.com/p/go-uuid/uuid"
)

var qs *QueryEngine

func TestMain(m *testing.M) {
	var err error
	qs, err = Open()
	if err != nil {
		log.Fatal(err)
	}
	defer qs.Close()
	os.Exit(m.Run())
}

func TestMutate(t *testing.T) {
	m := &schema.Mutation{
		ID:   uuid.NewUUID(),
		Name: "exampleOp",
		Args: []interface{}{
			"xxxxx",
			map[string]interface{}{
				"a": "thing",
				"b": 2,
			},
			123,
		},
	}
	err := qs.Mutate(m)
	if err != nil {
		t.Fatal(err)
	}
}

func TestQuery(t *testing.T) {
	res, err := qs.QueryBytes(`
    root() {
      friends() {
        first_name,
        last_name,
        age,
      }
    }
  `)
	if err != nil {
		t.Fatal(err)
	}

}
