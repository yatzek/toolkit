package main

import (
	"fmt"
	"github.com/yatzek/toolkit"
	"log"
	"net/http"
)

func main() {
	mux := routes()
	log.Println("Starting server on port 8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}

func routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("."))))
	mux.HandleFunc("/upload", uploadFiles)
	mux.HandleFunc("/upload-one", uploadOneFile)

	return mux
}

func uploadFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	t := toolkit.Tools{
		MaxFileSize:      1024 * 1024 * 1024,
		AllowedFileTypes: []string{"image/jpeg", "image/png", "image/gif"},
	}

	files, err := t.UploadFiles(r, "./uploads")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	out := ""
	for _, file := range files {
		out += fmt.Sprintf(
			"Uplaoded %s to the uploads folder, renamed to %s\n",
			file.OriginalFileName,
			file.NewFileName,
		)
	}
	_, _ = w.Write([]byte(out))
}

func uploadOneFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	t := toolkit.Tools{
		MaxFileSize:      1024 * 1024 * 1024,
		AllowedFileTypes: []string{"image/jpeg", "image/png", "image/gif"},
	}

	file, err := t.UploadOneFile(r, "./uploads")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, _ = w.Write([]byte(fmt.Sprintf("Uploaded one file, %s to the uploads folder", file.OriginalFileName)))
}
