package main

import (
	"crypto"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

const port string = "8080"

var pairs = make(map[string]string)

func mainHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		id := r.URL.Path[1:]

		url, ok := getURL(id)
		if !ok {
			w.WriteHeader(404)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write([]byte("Error! wrong URL"))
			return
		}

		w.Header().Set("Location", url)
		w.WriteHeader(307)

	case http.MethodPost:
		body, _ := ioutil.ReadAll(r.Body)
		id := string(body)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		if len(id) > 2048 {
			w.WriteHeader(400)
			w.Write([]byte("Error! URL length more then 2048 cymbols"))
			return
		}

		url, ok := getURL(id)
		var err error
		if !ok {
			url, err = shorten(id)
			if err != nil {
				w.WriteHeader(400)
				w.Write([]byte("Create short URL error"))
				return
			}
		}

		w.WriteHeader(201)
		w.Write([]byte(url))

	default:
		http.Error(w, "Error! Wrong request method", http.StatusMethodNotAllowed)

	}
}

func getURL(id string) (string, bool) {
	if len(id) <= 0 {
		return "", false
	}
	url, ok := pairs[id]
	if !ok {
		return "", false
	}

	return url, true
}

func shorten(s string) (string, error) {
	hasher := crypto.MD5.New()
	if _, err := hasher.Write([]byte(s)); err != nil {
		return "", fmt.Errorf("URL encoding error: %v", err)
	}
	hash := string(hasher.Sum([]byte{}))
	hash = hash[:7]
	id := base64.StdEncoding.EncodeToString([]byte(hash))
	id = strings.ReplaceAll(id, "=", "")
	id = strings.ReplaceAll(id, "/", "_")

	pairs[id] = s

	return id, nil
}
func main() {
	http.HandleFunc("/", mainHandler)

	fmt.Println(" :", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
