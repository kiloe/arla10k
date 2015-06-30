package main

import (
	"arla/mutationstore"
	"arla/querystore"
	"log"
	"net/http"
)

type opts struct {
	ConfigPath string
}

func main() {
	// init query store
	qs := querystore.New(&querystore.Config{
		Path: "/app/index.js",
	})
	go func() {
		err := qs.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()
	// init action store
	ms, err := mutationstore.Open("/var/state/datastore")
	if err != nil {
		log.Fatal(err)
	}
	for m := range ms.Replay() {
		err := qs.Mutate(m)
		if err != nil {
			log.Fatal(err)
		}
	}
	// init ident store
	// add handler for query
	// add handler for mutations/actions
	// add handler for login
	// start http server
	log.Fatal(http.ListenAndServe(":80"), nil)
}
