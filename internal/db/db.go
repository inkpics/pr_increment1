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
	mp    map[string]map[string]string
	delmp map[string]map[string]string
	mux   sync.Mutex
	db    *sqlx.DB
}

type rec struct {
	Person string `db:"person"`
	Short  string `db:"short"`
	Long   string `db:"long"`
	Delete bool   `db:"del"`
}

var m = DBMap{
	mp:    make(map[string]map[string]string),
	delmp: make(map[string]map[string]string),
	mux:   sync.Mutex{},
	db:    nil,
}

var errPgDuplicateCode pq.ErrorCode = "23505"
var ErrorDuplicate = fmt.Errorf("duplicate record")

func Close(conn string) {
	if conn != "" {
		m.db.Close()
	}
}

func Ping(conn string) bool {
	if conn == "" {
		return false
	}

	if err := m.db.Ping(); err != nil {
		return false
	}

	return true
}

func readPg(conn string) error {
	var err error
	m.db, err = sqlx.Connect("postgres", conn)
	if err != nil {
		return err
	}

	m.db.MustExec(`
		CREATE TABLE IF NOT EXISTS links (
			person text,
			short text unique,
			long text,
			del bool
		);
	`)

	r := rec{}
	rows, err := m.db.Queryx("SELECT * FROM links")
	if err != nil {
		return err
	}
	for rows.Next() {
		err := rows.StructScan(&r)
		if err != nil {
			return err
		}
		if r.Delete {
			if len(m.delmp[r.Person]) == 0 {
				m.delmp[r.Person] = make(map[string]string)
			}
			m.delmp[r.Person][r.Short] = r.Long
		} else {
			if len(m.mp[r.Person]) == 0 {
				m.mp[r.Person] = make(map[string]string)
			}
			m.mp[r.Person][r.Short] = r.Long
		}
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}

func readFileStorage(fileStoragePath string) error {
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

func ReadStorage(fileStoragePath, conn string) error {
	if conn != "" {
		err := readPg(conn)
		if err != nil {
			return fmt.Errorf("read from db error: %w", err)
		}

		return nil
	}

	if fileStoragePath != "" {
		err := readFileStorage(fileStoragePath)
		if err != nil {
			return fmt.Errorf("read from file storage error: %w", err)
		}
	}

	return nil
}

func writePg(person, id, s string) error {
	_, err := m.db.Exec("INSERT INTO links VALUES ($1, $2, $3)", person, id, s)
	if err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == errPgDuplicateCode {
				return ErrorDuplicate
			}
		}
		return fmt.Errorf("pg error: %w", err)
	}
	return nil
}

func writeFileStorage(fileStoragePath string) error {
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

func WriteStorage(fileStoragePath, conn, person, id, s string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if len(m.mp[person]) == 0 {
		m.mp[person] = make(map[string]string)
	}
	m.mp[person][id] = s

	if conn != "" {
		err := writePg(person, id, s)
		if err != nil {
			return fmt.Errorf("db error: %w", err)
		}

		return nil
	}

	if fileStoragePath != "" {
		err := writeFileStorage(fileStoragePath)
		if err != nil {
			return fmt.Errorf("file storage error: %w", err)
		}
	}

	return nil
}
func IDReadURL(person, id string) (string, int) {
	if len(id) <= 0 {
		return "", 0
	}
	m.mux.Lock()
	defer m.mux.Unlock()

	for person := range m.mp {
		url, ok := m.mp[person][id]
		if ok {
			return url, 1
		}
	}

	for person := range m.delmp {
		url, ok := m.delmp[person][id]
		if ok {
			return url, 2
		}
	}

	return "", 0
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
func DeleteURL(conn, person, short string) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	link, ok := m.mp[person][short]
	if !ok {
		return fmt.Errorf("error: can't delete URL")
	}
	delete(m.mp[person], short)

	if len(m.delmp[person]) == 0 {
		m.delmp[person] = make(map[string]string)
	}
	m.delmp[person][short] = link

	if conn != "" {
		_, err := m.db.Exec("UPDATE links SET del = TRUE WHERE person = '$1' AND short = '$2'", person, short)
		if err != nil {
			return fmt.Errorf("db error: %w", err)
		}
		return nil
	}

	return nil
}
