package app

import (
	"crypto"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo"
)

var pairs = make(map[string]string)

func ShortenerStart(host, port string) {
	pairsStr, _ := ioutil.ReadFile("db")
	json.Unmarshal(pairsStr, &pairs)

	e := echo.New()
	e.POST("/", newLink)
	e.GET("/:id", getLink)

	e.Logger.Fatal(e.Start(host + ":" + port))
}

func newLink(c echo.Context) error {
	body, _ := ioutil.ReadAll(c.Request().Body)
	link := string(body)

	if len(link) > 2048 {
		return c.String(http.StatusBadRequest, "error, the link cannot be longer than 2048 characters")
	} else {
		_, err := url.ParseRequestURI(link)
		if err != nil {
			return c.String(http.StatusBadRequest, "error, the link is invalid")
		}
	}

	url, ok := getURL(link)
	var err error
	if !ok {
		url, err = shortener(link)
		if err != nil {
			return c.String(http.StatusBadRequest, "error, failed to create a shortened URL")
		}
	}

	return c.String(http.StatusCreated, url)
}

func getLink(c echo.Context) error {
	id := c.Param("id")

	url, ok := getURL(id)
	if !ok {
		return c.String(http.StatusNotFound, "error, there is no such link")
	}

	return c.Redirect(http.StatusTemporaryRedirect, url)
}

func getURL(id string) (string, bool) {
	if len(id) <= 0 {
		return "", false
	}
	url, ok := pairs[id]
	if !ok {
		return "", false
	}

	return url, true
}

func shortener(s string) (string, error) {
	h := crypto.MD5.New()
	if _, err := h.Write([]byte(s)); err != nil {
		return "", fmt.Errorf("abbreviation error URL: %v", err)
	}
	hash := string(h.Sum([]byte{}))
	hash = hash[len(hash)-5:]
	id := base64.StdEncoding.EncodeToString([]byte(hash))
	id = strings.ToLower(id)[:len(id)-1]
	id = strings.ReplaceAll(id, "/", "")
	id = strings.ReplaceAll(id, "=", "")

	pairs[id] = s

	jsonStr, _ := json.Marshal(pairs)
	ioutil.WriteFile("db", []byte(jsonStr), 0666)

	return "http://localhost:8080/" + id, nil
}
