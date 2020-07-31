package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
)

func Details(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, runtime.Version())

	paths, _ := filepath.Glob("/workspace/*")
	fmt.Fprintf(w, "/workspace contents: %v\n", paths)
}
