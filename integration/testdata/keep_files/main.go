package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, runtime.Version())

		paths, _ := filepath.Glob("/workspace/*")
		fmt.Fprintf(w, "/workspace contents: %v\n", paths)

		contents, _ := ioutil.ReadFile("./assets/some-file")
		fmt.Fprintf(w, "file contents: %s\n", string(contents))
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil))
}
