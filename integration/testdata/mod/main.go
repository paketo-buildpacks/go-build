package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(w, runtime.Version())

		paths, _ := filepath.Glob("/workspace/*")
		fmt.Fprintf(w, "/workspace contents: %v\n", paths)
	})

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}
