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

func ReadDB() error {

	mString, err := ioutil.ReadFile("db.txt")
	if err != nil {
		fmt.Println("error read saved data from file")
		return nil
	}
	if err == nil {
		m.mux.Lock()
		json.Unmarshal(mString, &m.mp)
		m.mux.Unlock()
		fmt.Println("data readed from saved file")
	}
	return err
}
func WriteDB(id string, s string) error {
	m.mux.Lock()
	m.mp[id] = s //доступ к мапе на запись ключа
	m.mux.Unlock()
	jsonStr, err := json.Marshal(m.mp)
	if err != nil {
		fmt.Println("json encoding error: ", err)
		return err
	}
	if err == nil {
		ioutil.WriteFile("db.txt", []byte(jsonStr), 0666) //запись мапы в файл
		return nil
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
