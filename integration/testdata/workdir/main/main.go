package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	fmt.Print("Hello from subdir!\n")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello from subdir!")
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil))
}
