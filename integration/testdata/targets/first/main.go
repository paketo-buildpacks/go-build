package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/sahilm/fuzzy"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "first: %s\n", runtime.Version())
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil))

	// Useless code that adds an import
	pattern := "buildpacks"
	data := []string{"paketo", "buildpacks"}

	matches := fuzzy.Find(pattern, data)
	fmt.Println(len(matches))
}
