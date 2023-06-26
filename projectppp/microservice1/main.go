package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

var temp *template.Template

const chunkSize = 1024 * 1024 // 1MB

func init() {
	temp = template.Must(template.ParseFiles("template/index.html"))
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Provide the path to the service account JSON key file
	keyFilePath := "C:/Users/Admin/Desktop/WorkspaceGo/seismic-monitor-388610-52d860ffe0ef.json"

	// Create options with the credentials file
	opts := option.WithCredentialsFile(keyFilePath)

	// Create a new client for Google Cloud Storage with the options
	client, err := storage.NewClient(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Get the file from the request
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get the bucket handle
	bucketName := "chunk_bucket"
	bucket := client.Bucket(bucketName)

	// Create the object in the bucket
	objName := header.Filename
	obj := bucket.Object(objName)
	objWriter := obj.NewWriter(ctx)
	defer objWriter.Close()

	// Copy the file data to the object in chunks
	buf := make([]byte, chunkSize)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if n == 0 {
			break
		}

		// Write the chunk to the object
		if _, err := objWriter.Write(buf[:n]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Set the appropriate metadata for the object (optional)
	// You can modify this section as per your requirements
	objAttrs := storage.ObjectAttrsToUpdate{}
	objAttrs.ContentType = "application/octet-stream" // Set the content type if known

	// Check if the object already exists
	_, err = obj.Attrs(ctx)
	if err != nil {
		// Object doesn't exist, so create it with the specified metadata
		err = objWriter.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Object already exists, update the metadata
		_, err = obj.Update(ctx, objAttrs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	fmt.Fprintf(w, "File uploaded successfully!")
}

func HandleFunc(w http.ResponseWriter, r *http.Request) {
	http.HandleFunc("/upload", uploadHandler)
	temp.ExecuteTemplate(w, "index.html", nil)
}

func main() {
	http.HandleFunc("/", HandleFunc)
	//http.ListenAndServe(":9999", nil)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
