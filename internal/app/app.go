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

	// "sync"

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

func ShortenerInit() {

	err := db.ReadDB()
	if err != nil {
		fmt.Println("error read saved data from file")
		log.Panic(err)
	}

	r := chi.NewRouter()
	r.Post("/", createURL)
	r.Get("/{id}", receiveURL)
	log.Fatal(http.ListenAndServe(":8080", r))
}

func createURL(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	link := string(body)
	var err error
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
	w.Write([]byte(url))
}
func createJSONURL(w http.ResponseWriter, r *http.Request) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	link := string(body)
	JSONlink := make(map[string]string)
	err = json.Unmarshal(body, &JSONlink)
	if err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}

	link, ok := JSONlink["url"]
	if !ok {
		return fmt.Errorf("error: no such link")
	}
	if len(link) > 2048 {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error: the link cannot be longer than 2048 characters"))
		return fmt.Errorf("error: the link is invalid")
	}
	_, err = url.ParseRequestURI(link)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error: the link is invalid"))
		return fmt.Errorf("error: the link is invalid: %w", err)
	}

	url, ok := db.IDReadURL(link)
	if !ok {
		url, err = shortener(link)
		if err != nil {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("error: failed to create a shortened URL"))
			return fmt.Errorf("error: failed to create a shortened URL: %w", err)
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(url))
	return nil
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
	err := db.WriteDB(id, s)
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
	// 	ioutil.WriteFile("m.txt", []byte(jsonStr), 0666) //запись мапы в файл
	// }

	return "http://localhost:8080/" + id, nil
}
