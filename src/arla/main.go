package main

import (
	"arla/identstore"
	"arla/mutationstore"
	"arla/querystore"
	"arla/schema"
	"fmt"
	"log"
	"net/http"
	"time"
)

type opts struct {
	ConfigPath string
}

func main() {
	// init query store
	fmt.Println("starting query service...")
	qs, err := querystore.New(&querystore.Config{
		Path:     "/app/index.js",
		LogLevel: querystore.LOG,
	})
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		if err := qs.Wait(); err != nil {
			log.Fatal("qs died! ", err)
		}
	}()
	// init action store
	fmt.Println("starting mutation service...")
	ms, err := mutationstore.Open("/var/state/datastore")
	if err != nil {
		log.Fatal("ms open", err)
	}
	// silence logs for a bit and replay
	fmt.Println("replaying mutations...")
	start := time.Now()
	oldLogLevel := qs.GetLogLevel()
	qs.SetLogLevel(querystore.ERROR)
	w, err := qs.NewWriter()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := ms.WriteTo(w); err != nil {
		log.Fatal(err)
	}
	w.Close()
	qs.SetLogLevel(oldLogLevel)
	fmt.Printf("%d mutations replayed in %s\n", ms.Len(), time.Since(start))
	// init ident store
	fmt.Println("starting indent service...")
	is, err := identstore.Open("/var/state/ident")
	if err != nil {
		log.Fatal("is open ", err)
	}
	u := &schema.User{
		ID:      schema.MustParseUUID("f3817582-1f2d-11e5-a248-0242ac110001"),
		Aliases: []string{"admin", "administrator", "su", "superuser", "god"},
	}
	u.SetPassword("test")
	if err := is.Put(u); err != nil {
		log.Fatal("is put ", u)
	}

	// add handler for auth
	// listen at /sessions/
	// pass body into schema.AuthRequest
	// return token
	// add handler for mutations/actions
	// listen at /mutations/
	// check token
	// parse body into schema.Mutation
	// runs qs.Mutate(m)
	// logs successful modifications to ms.Write(m)
	// return OK / FAIL
	// add handler for query
	// listen at /query/
	// check token
	// parse body into schema.Query
	// run qs.Query
	// return response json / FAIL
	// start http server
	log.Println("arla is ready")
	fs := http.FileServer(http.Dir("/app/public"))
	http.Handle("/", fs)

	http.ListenAndServe(":80", nil)
}
