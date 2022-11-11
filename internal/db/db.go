package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type DBMap struct {
	mp  map[string]map[string]string
	mux sync.Mutex
}

type rec struct {
	Person string `db:"person"`
	Short  string `db:"short"`
	Long   string `db:"long"`
}

var m = DBMap{
	mp:  make(map[string]map[string]string),
	mux: sync.Mutex{},
}

var ErrorDuplicate = fmt.Errorf("duplicate record")

func ReadDB(fileStoragePath string, conn string) error {
	if conn != "" {
		db, err := sqlx.Connect("postgres", conn)
		if err != nil {
			return err
		}
		defer db.Close()

		db.MustExec(`
			CREATE TABLE IF NOT EXISTS links (
				person text,
				short text unique,
				long text
			);
		`)

		r := rec{}
		rows, err := db.Queryx("SELECT * FROM links")
		if err != nil {
			return err
		}
		for rows.Next() {
			err := rows.StructScan(&r)
			if err != nil {
				return err
			}
			if len(m.mp[r.Person]) == 0 {
				m.mp[r.Person] = make(map[string]string)
			}
			m.mp[r.Person][r.Short] = r.Long
		}
		err = rows.Err()
		if err != nil {
			return err
		}

		return nil
	}

	if fileStoragePath != "" {
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
	}

	return nil
}
func WriteDB(fileStoragePath string, conn string, person string, id string, s string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if len(m.mp[person]) == 0 {
		m.mp[person] = make(map[string]string)
	}
	m.mp[person][id] = s

	if conn != "" {
		db, err := sqlx.Connect("postgres", conn)
		if err != nil {
			return fmt.Errorf("db error: %w", err)
		}
		defer db.Close()

		_, err = db.Exec("INSERT INTO links VALUES ($1, $2, $3)", person, id, s)
		if err != nil {
			if err, ok := err.(*pq.Error); ok {
				if err.Code == "23505" {
					return ErrorDuplicate
				}
			}
			return fmt.Errorf("db error: %w", err)
		}

		return nil
	}

	if fileStoragePath != "" {
		jsonStr, err := json.Marshal(m.mp)
		if err != nil {
			return fmt.Errorf("json encoding error: %w", err)
		}
		err = os.WriteFile(fileStoragePath, jsonStr, 0666) //запись мапы в файл
		if err != nil {
			return fmt.Errorf("write to file error: %w", err)
		}
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
