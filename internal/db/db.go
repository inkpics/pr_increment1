package db

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

type DBMap struct {
	mp  map[string]string
	mux sync.Mutex
}

var m = DBMap{
	mp:  make(map[string]string),
	mux: sync.Mutex{},
}

const defaultDbPath = "./db.txt"

func ReadDB(fileStoragePath string) error {
	if fileStoragePath == "" {
		fileStoragePath = defaultDbPath
	}

	mString, err := os.ReadFile(fileStoragePath)
	if err != nil {
		dbFileNew, err := os.Create(fileStoragePath)
		if err != nil {
			log.Fatal(err)
		}
		dbFileNew.Close()

		mString, err = os.ReadFile(fileStoragePath)
		if err != nil {
			log.Fatal(err)
		}

	}

	m.mux.Lock()
	err = json.Unmarshal(mString, &m.mp)
	m.mux.Unlock()
	if err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}
	return nil
}
func WriteDB(fileStoragePath string, id string, s string) error {
	m.mux.Lock()
	m.mp[id] = s //доступ к мапе на запись ключа
	m.mux.Unlock()
	jsonStr, err := json.Marshal(m.mp)
	if err != nil {
		return fmt.Errorf("json encoding error: %w", err)
	}
	err = os.WriteFile(fileStoragePath, []byte(jsonStr), 0666) //запись мапы в файл
	if err != nil {
		return fmt.Errorf("data write to file error: %w", err)
	}
	return nil
}
func IDReadURL(id string) (string, bool) {
	if len(id) <= 0 {
		return "", false
	}
	m.mux.Lock()
	url, ok := m.mp[id] // доступ к мапе на чтение URL по ключу
	m.mux.Unlock()
	if !ok {
		return "", false
	}

	return url, true
}
