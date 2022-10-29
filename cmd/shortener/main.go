package main

import (
	"flag"
	"os"

	"github.com/inkpics/pr_increment1/internal/app"
)

const baseServerAddress = "localhost:8080"

func main() {
	var serverAddress string
	var baseURL string
	var fileStoragePath string

	flag.StringVar(&serverAddress, "a", os.Getenv("SERVER_ADDRESS"), "server adress")
	flag.StringVar(&baseURL, "b", os.Getenv("BASE_URL"), "URL")
	flag.StringVar(&fileStoragePath, "f", os.Getenv("FILE_STORAGE_PATH"), "data file storage path")
	flag.Parse()

	if serverAddress == "" {
		serverAddress = baseServerAddress
	}

	if baseURL == "" {
		baseURL = "http://" + serverAddress
	}
	app.ShortenerInit(serverAddress, baseURL, fileStoragePath)
}
