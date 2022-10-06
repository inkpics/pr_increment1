package db

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

func ReadDB(FILE_STORAGE_PATH string) error {
	if FILE_STORAGE_PATH == "" {
		return nil
	}

	mString, err := ioutil.ReadFile(FILE_STORAGE_PATH)
	if err != nil {
		fmt.Println("error read saved data from file")
		return nil
	}

	m.mux.Lock()
	err = json.Unmarshal(mString, &m.mp)
	m.mux.Unlock()
	if err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}
	fmt.Println("data readed from saved file")
	return nil
}
func WriteDB(FILE_STORAGE_PATH string, id string, s string) error {
	m.mux.Lock()
	m.mp[id] = s //доступ к мапе на запись ключа
	m.mux.Unlock()
	jsonStr, err := json.Marshal(m.mp)
	if err != nil {
		return fmt.Errorf("json encoding error: %w", err)
	}
	ioutil.WriteFile(FILE_STORAGE_PATH, []byte(jsonStr), 0666) //запись мапы в файл
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
