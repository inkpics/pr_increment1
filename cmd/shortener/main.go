package main

import (
	"flag"
	"os"

	"github.com/inkpics/pr_increment1/internal/app"
)

func main() {
	var serverAddress string
	var baseURL string
	var fileStoragePath string

	flag.StringVar(&serverAddress, "a", os.Getenv("SERVER_ADDRESS"), "SERVER_ADDRESS")
	flag.StringVar(&baseURL, "b", os.Getenv("BASE_URL"), "BASE_URL")
	flag.StringVar(&fileStoragePath, "f", os.Getenv("FILE_STORAGE_PATH"), "FILE_STORAGE_PATH")
	flag.Parse()

	app.ShortenerInit(serverAddress, baseURL, fileStoragePath)
}
