package main

import (
	"flag"
	"os"

	"github.com/inkpics/pr_increment1/internal/app"
)

func main() {
	var SERVER_ADDRESS string
	var BASE_URL string
	var FILE_STORAGE_PATH string

	flag.StringVar(&SERVER_ADDRESS, "a", os.Getenv("SERVER_ADDRESS"), "SERVER_ADDRESS")
	flag.StringVar(&BASE_URL, "b", os.Getenv("BASE_URL"), "BASE_URL")
	flag.StringVar(&FILE_STORAGE_PATH, "f", os.Getenv("FILE_STORAGE_PATH"), "FILE_STORAGE_PATH")
	flag.Parse()

	app.ShortenerInit(SERVER_ADDRESS, BASE_URL, FILE_STORAGE_PATH)
}
