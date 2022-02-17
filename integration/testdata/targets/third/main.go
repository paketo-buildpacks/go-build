package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/Masterminds/semver"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "third: %s\n", runtime.Version())
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil))

	// Useless code that adds an import
	v, _ := semver.NewVersion("1.2.3-beta.1+build345")
	fmt.Println(v.String())
}
