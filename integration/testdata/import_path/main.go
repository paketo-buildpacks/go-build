package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/paketo-buildpacks/go-build/integration/testdata/import_path/handlers"
)

func main() {
	http.HandleFunc("/", handlers.Details)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil))
}
