package app

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetURL(t *testing.T) {
	tests := []struct {
		long  string
		short string
	}{
		{long: "http://yandex.ru", short: "obrnfe4"},
		{long: "http://yandex.ru/", short: "xtklsi"},
		{long: "http://praktikum.yandex.ru", short: "5te+dbq"},
		{long: "http://maps.yandex.ru", short: "lbohctw"},
		{long: "", short: "moz4qn4"},
		{long: "//test_link", short: "pf7smo8"},
	}

	for _, testCase := range tests {
		short, err := shortener(testCase.long)
		if err != nil {
			t.Errorf("can't shorten link %v", testCase.long)
		}

		if short != "http://localhost:8080/"+testCase.short {
			t.Fatalf("expected short link %v; got %v", "http://localhost:8080/"+testCase.short, short)
		}

		long, _ := getURL(testCase.short)
		if long != testCase.long {
			t.Fatalf("expected long link %v; got %v", testCase.long, long)
		}
	}
}

func TestShortener(t *testing.T) {
	tests := []struct {
		long  string
		short string
	}{
		{long: "http://yandex.ru", short: "obrnfe4"},
		{long: "http://yandex.ru/", short: "xtklsi"},
		{long: "http://praktikum.yandex.ru", short: "5te+dbq"},
		{long: "http://maps.yandex.ru", short: "lbohctw"},
		{long: "", short: "moz4qn4"},
		{long: "//test_link", short: "pf7smo8"},
	}

	for _, testCase := range tests {
		short, err := shortener(testCase.long)
		if err != nil {
			t.Errorf("can't shorten link %v", testCase.long)
		}

		if short != "http://localhost:8080/"+testCase.short {
			t.Fatalf("expected short link %v; got %v", "http://localhost:8080/"+testCase.short, short)
		}
	}
}

func TestMainHandler(t *testing.T) {
	tests := []struct {
		long  string
		short string
	}{
		{long: "http://yandex.ru", short: "obrnfe4"},
		{long: "http://yandex.ru/", short: "xtklsi"},
		{long: "http://praktikum.yandex.ru", short: "5te+dbq"},
		{long: "http://maps.yandex.ru", short: "lbohctw"},
		{long: "", short: "moz4qn4"},
		{long: "//test_link", short: "pf7smo8"},
	}

	{
		request, err := http.NewRequest("PUT", "localhost:8080", bytes.NewReader([]byte("http://yandex.ru")))
		if err != nil {
			t.Fatalf("could not create request: %v", err)
		}

		record := httptest.NewRecorder()
		mainHandler(record, request)

		result := record.Result()
		defer result.Body.Close()

		if result.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected status %v; got %v", http.StatusMethodNotAllowed, result.StatusCode)
		}
	}

	{
		request, err := http.NewRequest("GET", "localhost:8080", nil)
		if err != nil {
			t.Fatalf("could not create request: %v", err)
		}

		record := httptest.NewRecorder()
		mainHandler(record, request)

		result := record.Result()
		defer result.Body.Close()

		if result.StatusCode != http.StatusNotFound {
			t.Errorf("expected status %v; got %v", http.StatusNotFound, result.StatusCode)
		}
	}

	{
		request, err := http.NewRequest("POST", "localhost:8080", bytes.NewReader([]byte(strings.Repeat("A", 2049))))
		if err != nil {
			t.Fatalf("could not create request: %v", err)
		}

		record := httptest.NewRecorder()
		mainHandler(record, request)

		result := record.Result()
		defer result.Body.Close()

		if result.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status %v; got %v", http.StatusBadRequest, result.StatusCode)
		}
	}

	{
		request, err := http.NewRequest("POST", "localhost:8080", bytes.NewReader([]byte("123456")))
		if err != nil {
			t.Fatalf("could not create request: %v", err)
		}
		record := httptest.NewRecorder()
		mainHandler(record, request)

		result := record.Result()
		defer result.Body.Close()

		if result.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status %v; got %v", http.StatusBadRequest, result.StatusCode)
		}
	}

	for _, testCase := range tests {
		if testCase.long == "" {
			continue
		}

		request, err := http.NewRequest("POST", "localhost:8080", bytes.NewReader([]byte(testCase.long)))
		if err != nil {
			t.Fatalf("could not create request: %v", err)
		}

		record := httptest.NewRecorder()
		mainHandler(record, request)

		result := record.Result()
		defer result.Body.Close()

		if result.StatusCode != http.StatusCreated {
			t.Errorf("expected status %v; got %v", http.StatusCreated, result.StatusCode)
		}

		body, err := ioutil.ReadAll(result.Body)
		if err != nil {
			t.Fatalf("could not read response: %v", err)
		}

		short := string(body)
		if short != "http://localhost:8080/"+testCase.short {
			t.Fatalf("expected answer to be %v; got %v", "http://localhost:8080/"+testCase.short, short)
		}
	}

}
