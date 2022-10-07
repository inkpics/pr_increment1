package app

import (
	"compress/gzip"
	"crypto"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	// "sync"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/inkpics/pr_increment1/internal/db"
)

// type SafeMap struct {
// 	mp  map[string]string
// 	mux sync.Mutex
// }

// var m = SafeMap{
// 	mp:  make(map[string]string),
// 	mux: sync.Mutex{},
// }

var (
	fsPath string
	base   string
)

type Res struct {
	Result string `json:"result"`
}

func ShortenerInit(serverAddress, baseURL, fileStoragePath string) {
	fsPath = fileStoragePath
	base = baseURL

	err := db.ReadDB(fsPath)
	if err != nil {
		fmt.Println("error read saved data from file")
		log.Panic(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Compress(5))
	r.Post("/", createURL)
	r.Post("/api/shorten", createJSONURL)
	r.Get("/{id}", receiveURL)
	log.Fatal(http.ListenAndServe(serverAddress, r))
}

func createURL(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("read body")
		return
	}
	link := string(body)

	if r.Header.Get("Content-Encoding") == "gzip" || r.Header.Get("Content-Encoding") == "x-gzip" {
		body, err = readAll(r.Body)
		if err != nil {
			fmt.Println("read body gzip")
			return
		}
	}

	if len(link) > 2048 {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error: the link cannot be longer than 2048 characters"))
		return
	}
	_, err = url.ParseRequestURI(link)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error: the link is invalid"))
		return
	}

	url, ok := db.IDReadURL(link)
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
	w.Write([]byte(base + "/" + url))
}
func createJSONURL(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("read body")
		return
	}
	if r.Header.Get("Content-Encoding") == "gzip" || r.Header.Get("Content-Encoding") == "x-gzip" {
		body, err = readAll(r.Body)
		if err != nil {
			fmt.Println("read body gzip")
			return
		}
	}
	JSONlink := make(map[string]string)
	err = json.Unmarshal(body, &JSONlink)
	if err != nil {
		fmt.Println("json unmarshal error")
		return
	}

	link, ok := JSONlink["url"]
	if !ok {
		fmt.Println("error: no such link")
		return
	}
	if len(link) > 2048 {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error: the link cannot be longer than 2048 characters"))
		fmt.Println("error: the link cannot be longer than 2048 characters")
		return
	}

	_, err = url.ParseRequestURI(link)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error: the link is invalid"))
		fmt.Println("error: the link is invalid")
		return
	}

	url, ok := db.IDReadURL(link)
	if !ok {
		url, err = shortener(link)
		if err != nil {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("error: failed to create a shortened URL"))
			fmt.Println("error: failed to create a shortened URL")
			return
		}
	}

	result := &Res{
		Result: base + "/" + url,
	}

	jsonStr, err := json.Marshal(result)
	if err != nil {
		fmt.Println("json encoding error")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(jsonStr))
	//return
}

func receiveURL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	url, ok := db.IDReadURL(id)
	if !ok {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("error: there is no such link"))
		return
	}

	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// func IDReadURL(id string) (string, bool) {
// 	if len(id) <= 0 {
// 		return "", false
// 	}
// 	m.mux.Lock()
// 	url, ok := m.mp[id] // доступ к мапе на чтение URL по ключу
// 	m.mux.Unlock()
// 	if !ok {
// 		return "", false
// 	}

// 	return url, true
// }

func shortener(s string) (string, error) {
	h := crypto.MD5.New()
	if _, err := h.Write([]byte(s)); err != nil {
		return "", fmt.Errorf("abbreviation  URL error: %w", err)
	}
	hash := string(h.Sum([]byte{}))
	hash = hash[len(hash)-5:]
	id := base64.StdEncoding.EncodeToString([]byte(hash))
	id = strings.ToLower(id)[:len(id)-1]
	id = strings.ReplaceAll(id, "/", "")
	id = strings.ReplaceAll(id, "=", "")
	err := db.WriteDB(fsPath, id, s)
	if err != nil {
		return "", fmt.Errorf("error write data to file: %w", err)
	}
	// m.mux.Lock()
	// m.mp[id] = s //доступ к мапе на запись ключа
	// m.mux.Unlock()
	// jsonStr, ok := json.Marshal(m.mp)
	// if ok != nil {
	// 	fmt.Println("json encoding error: ", ok)
	// }
	// if ok == nil {
	// 	io.WriteFile("m.txt", []byte(jsonStr), 0666) //запись мапы в файл
	// }

	return id, nil
}

func readAll(r io.Reader) ([]byte, error) {
	reader, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	buff, err := io.ReadAll(reader)
	return buff, err
}