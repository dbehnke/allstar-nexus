package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
)

//go:embed all:frontend/out
var frontendFiles embed.FS

func main() {
	// Create a sub-filesystem for the frontend/out directory
	sub, err := fs.Sub(frontendFiles, "frontend/out")
	if err != nil {
		fmt.Printf("Error creating sub-filesystem: %s\n", err)
		os.Exit(1)
	}

	// Create a file server for the embedded frontend
	frontendHandler := http.FileServer(http.FS(sub))
	http.Handle("/", frontendHandler)

	fmt.Println("Starting server on :8080, serving the Next.js frontend.")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
		os.Exit(1)
	}
}
