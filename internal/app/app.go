package app

import (
	"crypto"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

var m = make(map[string]string)

func ShortenerInit() {
	mStr, _ := ioutil.ReadFile("m.txt")
	json.Unmarshal(mStr, &m)

	r := chi.NewRouter()
	r.Post("/", createURL)
	r.Get("/{id}", receiveURL)
	log.Fatal(http.ListenAndServe(":8080", r))
}

func createURL(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	link := string(body)

	if len(link) > 2048 {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error: the link cannot be longer than 2048 characters"))
		return
	} else {
		_, err := url.ParseRequestURI(link)
		if err != nil {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("error: the link is invalid"))
			return
		}
	}

	url, ok := getURL(link)
	var err error
	if !ok {
		url, err = shortener(link)
		if err != nil {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("error: failed to create a shortened URL"))
			return
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(url))
	return
}

func receiveURL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	url, ok := getURL(id)
	if !ok {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("error: there is no such link"))
		return
	}

	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
	return
}

func getURL(id string) (string, bool) {
	if len(id) <= 0 {
		return "", false
	}
	url, ok := m[id]
	if !ok {
		return "", false
	}

	return url, true
}

func shortener(s string) (string, error) {
	h := crypto.MD5.New()
	if _, err := h.Write([]byte(s)); err != nil {
		return "", fmt.Errorf("abbreviation error URL: %v", err)
	}
	hash := string(h.Sum([]byte{}))
	hash = hash[len(hash)-5:]
	id := base64.StdEncoding.EncodeToString([]byte(hash))
	id = strings.ToLower(id)[:len(id)-1]
	id = strings.ReplaceAll(id, "/", "")
	id = strings.ReplaceAll(id, "=", "")

	m[id] = s

	jsonStr, _ := json.Marshal(m)
	ioutil.WriteFile("m.txt", []byte(jsonStr), 0666)

	return "http://localhost:8080/" + id, nil
}
