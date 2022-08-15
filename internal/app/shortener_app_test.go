package app

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	tests = []struct {
		long  string
		short string
	}{
		{long: "http://yandex.ru", short: "obrnfe4"},
		{long: "http://praktikum.yandex.ru", short: "5te+dbq"},
		{long: "http://direct.yandex.ru", short: "jvbrdoy"},
	}
)

func TestGetURL(t *testing.T) {
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

func TestCreateURL(t *testing.T) {
	{
		request, _ := http.NewRequest(http.MethodPost, "localhost:8080", strings.NewReader(strings.Repeat("A", 2049)))

		recorder := httptest.NewRecorder()
		createURL(recorder, request)

		result := recorder.Result()
		defer result.Body.Close()

		if result.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status %v; got %v", http.StatusBadRequest, result.StatusCode)
		}
	}

	for _, testCase := range tests {
		if testCase.long == "" {
			continue
		}

		request, _ := http.NewRequest(http.MethodPost, "localhost:8080", strings.NewReader(testCase.long))

		recorder := httptest.NewRecorder()
		createURL(recorder, request)

		result := recorder.Result()
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

func TestReceiveURL(t *testing.T) {
	request, _ := http.NewRequest(http.MethodGet, "localhost:8080", nil)

	recorder := httptest.NewRecorder()
	receiveURL(recorder, request)

	result := recorder.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %v; got %v", http.StatusNotFound, result.StatusCode)
	}
}
