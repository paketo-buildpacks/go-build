package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/gorilla/mux"
)

//go:embed .occam-key
var s string

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(w, runtime.Version())
	})

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}
