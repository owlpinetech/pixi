package main

import (
	"flag"
	"log"
	"net/http"
)

func home(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello from Snippetbox"))
}

func main() {
	pixiDir := flag.String("pixiDir", "./", "the directory to serve Pixi files from")

	flag.Parse()

	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir(*pixiDir))

	mux.HandleFunc("/", home)
	mux.Handle("GET /pixi/", http.StripPrefix("/pixi", fileServer))

	log.Print("starting server on :4000")

	err := http.ListenAndServe(":4000", mux)
	log.Fatal(err)
}
