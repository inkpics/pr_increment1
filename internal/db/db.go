package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

type DBMap struct {
	mp  map[string]map[string]string
	mux sync.Mutex
}

var m = DBMap{
	mp:  make(map[string]map[string]string),
	mux: sync.Mutex{},
}

func ReadDB(fileStoragePath string) error {
	if fileStoragePath == "" {
		return nil
	}

	mString, err := os.ReadFile(fileStoragePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read from file error: %w", err)
	}

	m.mux.Lock()
	err = json.Unmarshal(mString, &m.mp)
	m.mux.Unlock()
	if err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}
	return nil
}
func WriteDB(fileStoragePath string, person string, id string, s string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if len(m.mp[person]) == 0 {
		m.mp[person] = make(map[string]string)
	}
	m.mp[person][id] = s

	if fileStoragePath == "" {
		return nil
	}

	jsonStr, err := json.Marshal(m.mp)
	if err != nil {
		return fmt.Errorf("json encoding error: %w", err)
	}
	err = os.WriteFile(fileStoragePath, jsonStr, 0666) //запись мапы в файл
	if err != nil {
		return fmt.Errorf("write to file error: %w", err)
	}
	return nil
}
func IDReadURL(id string) (string, bool) {
	if len(id) <= 0 {
		return "", false
	}
	m.mux.Lock()
	defer m.mux.Unlock()

	for person := range m.mp {
		url, ok := m.mp[person][id]
		if ok {
			return url, true
		}
	}

	return "", false
}
func ReceiveListURL(person string) (map[string]string, bool) {
	m.mux.Lock()
	defer m.mux.Unlock()

	lst, ok := m.mp[person]
	if !ok {
		return lst, false
	}

	return lst, true
}
