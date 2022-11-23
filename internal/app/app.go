package app

import (
	"crypto"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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

	err := db.ReadStorage(fsPath, conn)
	if err != nil {
		log.Panic(err)
	}
	defer db.Close(conn)

	e := echo.New()
	e.Use(middleware.Gzip())
	e.Use(middleware.Decompress())
	e.POST("/", createURL)
	e.POST("/api/shorten", createJSONURL)
	e.POST("/api/shorten/batch", createBatchJSONURL)
	e.GET("/:id", receiveURL)
	e.GET("/api/user/urls", receiveListURL)
	e.GET("/ping", ping)
	e.DELETE("/api/user/urls", deleteListURL)
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

	person := checkPerson(c, enc)
	url, variant := db.IDReadURL(person, link)
	if variant == 0 {
		url, err = shortener(link, person)
		if errors.Is(err, db.ErrorDuplicate) {
			return c.String(http.StatusConflict, base+"/"+url)
		}
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

	person := checkPerson(c, enc)
	url, variant := db.IDReadURL(person, link)
	if variant == 0 {
		url, err = shortener(link, person)
		if errors.Is(err, db.ErrorDuplicate) {
			result := &res{
				Result: base + "/" + url,
			}
			return c.JSON(http.StatusConflict, result)
		}
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
		return c.String(http.StatusBadRequest, "error: bad request")
	}

	var resp []batchResponse
	for result := range req {
		link := req[result]

		_, err = url.ParseRequestURI(link.OriginalURL)
		if err != nil {
			return c.String(http.StatusBadRequest, "error: the link is invalid")
		}

		person := checkPerson(c, enc)
		url, variant := db.IDReadURL(person, link.OriginalURL)
		if variant == 0 {
			url, err = shortener(link.OriginalURL, person)
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

	url, variant := db.IDReadURL(checkPerson(c, enc), id)
	if variant == 0 {
		return c.String(http.StatusNotFound, "error: there is no such link")
	} else if variant == 2 {
		return c.String(http.StatusGone, "error: link has been deleted")
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

func deleteListURL(c echo.Context) error {
	person := checkPerson(c, enc)
	list, ok := db.ReceiveListURL(person)
	if !ok {
		return c.String(http.StatusBadRequest, "error: person have no links")
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusBadRequest, "error: failed to get links to delete")
	}

	var links []string
	err = json.Unmarshal(body, &links)
	if err != nil {
		return c.String(http.StatusBadRequest, "error: failed to unmarshal links")
	}

	for _, del := range links {
		for cur := range list {
			if cur == del {
				go db.DeleteURL(conn, person, del)
				break
			}
		}
	}

	return c.String(http.StatusAccepted, "URLs deleted")
}

func ping(c echo.Context) error {
	if ok := db.Ping(conn); !ok {
		return c.String(http.StatusInternalServerError, "error: db is not active")
	}

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

func signition(person, enc string) string {
	hm := hmac.New(sha256.New, []byte(enc))
	hm.Write([]byte(person))
	result := hm.Sum(nil)
	return hex.EncodeToString(result)[:16]
}

func cookie(c echo.Context, name, val string) (string, error) {
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

func shortener(s, person string) (string, error) {
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
	err := db.WriteStorage(fsPath, conn, person, id, s)
	if errors.Is(err, db.ErrorDuplicate) {
		return id, db.ErrorDuplicate
	}
	if err != nil {
		return "", fmt.Errorf("error write data to file: %w", err)
	}

	return id, nil
}
