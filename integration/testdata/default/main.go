package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

// Embeds the .occam-key to make the images unique after the source is removed.
//go:embed .occam-key
var s string

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, runtime.Version())

		paths, _ := filepath.Glob("/workspace/*")
		fmt.Fprintf(w, "/workspace contents: %v\n", paths)
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil))
}
