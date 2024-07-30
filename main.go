package main

import (
	"fmt"
	"net/http"
	"log"
)

func main() {
	addr := "192.168.0.26:8080"

	mux := http.NewServeMux()

	httpServer := &http.Server{
		Addr: addr,
		Handler: mux,
	}

	mux.Handle("/", http.FileServer(http.Dir("site")))

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
	})

	fmt.Println("Server started at " + addr)

	log.Fatal(httpServer.ListenAndServe())
}
