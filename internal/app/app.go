package app

import (
	"crypto"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/inkpics/pr_increment1/internal/db"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const intLeng = 2048

var (
	fsPath string
	base   string
)

type res struct {
	Result string `json:"result"`
}

func ShortenerInit(serverAddress, baseURL, fileStoragePath string) {
	fsPath = fileStoragePath
	base = baseURL

	err := db.ReadDB(fsPath)
	if err != nil {
		log.Panic(err)
	}

	e := echo.New()
	e.Use(middleware.Gzip())
	e.Use(middleware.Decompress())
	e.POST("/", createURL)
	e.POST("/api/shorten", createJSONURL)
	e.GET("/:id", receiveURL)
	e.Logger.Fatal(e.Start(serverAddress))
}

func createURL(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusBadRequest, "error: bad request")
	}
	link := string(body)

	if len(link) > intLeng {
		return c.String(http.StatusBadRequest, "error: the link cannot be longer than 2048 characters")
	}
	_, err = url.ParseRequestURI(link)
	if err != nil {
		return c.String(http.StatusBadRequest, "error: the link is invalid")
	}

	url, ok := db.IDReadURL(link)
	if !ok {
		url, err = shortener(link)
		if err != nil {
			return c.String(http.StatusBadRequest, "error: failed to create a shortened URL")
		}
	}

	return c.String(http.StatusCreated, base+"/"+url)
}
func createJSONURL(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusBadRequest, "error: bad request")
	}

	JSONlink := make(map[string]string)
	err = json.Unmarshal(body, &JSONlink)
	if err != nil {
		return c.String(http.StatusBadRequest, "error: bad request")
	}

	link, ok := JSONlink["url"]
	if !ok {
		return c.String(http.StatusBadRequest, "error: bad request")
	}
	if len(link) > intLeng {
		return c.String(http.StatusBadRequest, "error: the link cannot be longer than 2048 characters")
	}

	_, err = url.ParseRequestURI(link)
	if err != nil {
		return c.String(http.StatusBadRequest, "error: the link is invalid")
	}

	url, ok := db.IDReadURL(link)
	if !ok {
		url, err = shortener(link)
		if err != nil {
			return c.String(http.StatusBadRequest, "error: failed to create a shortened URL")
		}
	}

	result := &res{
		Result: base + "/" + url,
	}

	return c.JSON(http.StatusCreated, result)
}

func receiveURL(c echo.Context) error {
	id := c.Param("id")

	url, ok := db.IDReadURL(id)
	if !ok {
		return c.String(http.StatusNotFound, "error: there is no such link")
	}

	return c.Redirect(http.StatusTemporaryRedirect, url)
}

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
	err := db.WriteDB(fsPath, id, s)
	if err != nil {
		return "", fmt.Errorf("error write data to file: %w", err)
	}

	return id, nil
}
