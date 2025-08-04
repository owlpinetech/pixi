package main

import (
	"flag"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
)

// This is an example application showing that it is possible to serve Pixi files and easily read them using the
// Pixi library. It serves files from a specified folder and allows you to access them via HTTP.

func main() {
	port := flag.Int("port", 8080, "port to serve Pixi files on")
	folder := flag.String("folder", "./static", "folder to serve Pixi files from")
	flag.Parse()

	if *folder == "" {
		slog.Error("No folder specified to serve Pixi files from")
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/pixi/", handlePixi(*folder))

	slog.Info("Serving pixi files", "folder", *folder, "port", *port)
	err := http.ListenAndServe(":"+strconv.Itoa(*port), mux)
	if err != nil {
		slog.Error("Failed to start server", "error", err)
		return
	}
}

func handlePixi(folder string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get only the file name.
		filename := path.Base(r.URL.String())
		// Whatever arbitrary logic you want. For demo purposes, we will not serve
		// content that contains the string "bad". But you could be checking that
		// the user is the owner of this file, or any additional checks you aren't
		// doing in your middleware.
		// if strings.Contains(filename, "bad") {
		// 	fmt.Println("This file is bad and we won't serve it:", filename)
		// 	w.WriteHeader(http.StatusForbidden)
		// 	w.Write([]byte("Not authorized!"))
		// 	return
		// }
		// fmt.Println("Attempting to serve", filename)
		// Use http.ServeFile to serve the content. We don't have to worry about
		// writing not found HTTP error codes, etc., because ServeFile handles it
		// for us.
		http.ServeFile(w, r, filepath.Join(".", folder, filename))
	}
}
