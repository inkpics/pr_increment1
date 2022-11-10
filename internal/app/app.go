package app

import (
	"crypto"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	_ "github.com/lib/pq"

	"github.com/google/uuid"
	"github.com/inkpics/pr_increment1/internal/db"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const intLeng = 2048

var (
	fsPath string
	base   string
	conn   string
)

var enc = "secret"

type res struct {
	Result string `json:"result"`
}

type link struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type batchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type batchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

func ShortenerInit(serverAddress, baseURL, fileStoragePath, dbConn string) {
	fsPath = fileStoragePath
	base = baseURL
	conn = dbConn

	err := db.ReadDB(fsPath, conn)
	if err != nil {
		log.Panic(err)
	}

	e := echo.New()
	e.Use(middleware.Gzip())
	e.Use(middleware.Decompress())
	e.POST("/", createURL)
	e.POST("/api/shorten", createJSONURL)
	e.POST("/api/shorten/batch", createBatchJSONURL)
	e.GET("/:id", receiveURL)
	e.GET("/api/user/urls", receiveListURL)
	e.GET("/ping", ping)
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
		url, err = shortener(link, checkPerson(c, enc))
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
		url, err = shortener(link, checkPerson(c, enc))
		if err != nil {
			return c.String(http.StatusBadRequest, "error: failed to create a shortened URL")
		}
	}

	result := &res{
		Result: base + "/" + url,
	}

	return c.JSON(http.StatusCreated, result)
}

func createBatchJSONURL(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusBadRequest, "error: bad request")
	}

	var req []batchRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		return err
	}

	var resp []batchResponse
	for result := range req {
		link := req[result]

		_, err = url.ParseRequestURI(link.OriginalURL)
		if err != nil {
			return c.String(http.StatusBadRequest, "error: the link is invalid")
		}

		url, ok := db.IDReadURL(link.OriginalURL)
		if !ok {
			url, err = shortener(link.OriginalURL, checkPerson(c, enc))
			if err != nil {
				return c.String(http.StatusBadRequest, "error: failed to create a shortened URL")
			}
		}

		resp = append(resp, batchResponse{
			CorrelationID: link.CorrelationID,
			ShortURL:      base + "/" + url,
		})
	}

	return c.JSON(http.StatusCreated, resp)
}

func receiveURL(c echo.Context) error {
	id := c.Param("id")

	url, ok := db.IDReadURL(id)
	if !ok {
		return c.String(http.StatusNotFound, "error: there is no such link")
	}

	return c.Redirect(http.StatusTemporaryRedirect, url)
}

func receiveListURL(c echo.Context) error {
	person := checkPerson(c, enc)
	list, ok := db.ReceiveListURL(person)
	if !ok {
		return c.String(http.StatusNoContent, "error: person have no links")
	}

	var persLinks []*link
	for short, orig := range list {
		persLinks = append(persLinks, &link{
			ShortURL:    base + "/" + short,
			OriginalURL: orig,
		})
	}

	return c.JSON(http.StatusOK, persLinks)
}

func ping(c echo.Context) error {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return c.String(http.StatusInternalServerError, "error: db is not active")
	}
	defer db.Close()

	return c.String(http.StatusOK, "db is active")
}

func checkPerson(c echo.Context, enc string) string {
	person, err0 := cookie(c, "person", "")
	token, err1 := cookie(c, "token", "")
	if err0 == nil && err1 == nil && token == signition(person, enc) {
		return person
	}

	person = uuid.New().String()
	cookie(c, "person", person)
	cookie(c, "token", signition(person, enc))
	return person
}

func signition(person string, enc string) string {
	hm := hmac.New(sha256.New, []byte(enc))
	hm.Write([]byte(person))
	result := hm.Sum(nil)
	return hex.EncodeToString(result)[:16]
}

func cookie(c echo.Context, name string, val string) (string, error) {
	coo := new(http.Cookie)

	if val == "" {
		coo, err := c.Cookie(name)
		if err != nil {
			return "", err
		}
		return coo.Value, nil
	}

	coo.Name = name
	coo.Value = val
	c.SetCookie(coo)
	return "", nil
}

func shortener(s string, person string) (string, error) {
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
	err := db.WriteDB(fsPath, conn, person, id, s)
	if err != nil {
		return "", fmt.Errorf("error write data to file: %w", err)
	}

	return id, nil
}
