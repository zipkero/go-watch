package main

import (
	"fmt"
	"github.com/zipkero/go-watch/internal/watcher"
	"log"
	"net/url"
	"time"
)

var (
	targetUrl   string
	requests    int
	concurrency int
	delay       int
)

func main() {
	setInputParams()
	wc := watcher.NewWatcher(targetUrl, requests, concurrency, time.Duration(delay))
	if wc == nil {
		panic("Watcher is nil")
	}
	wc.Start()
}

func setInputParams() {
	fmt.Print("Enter URL: ")
	var err error
	_, err = fmt.Scanln(&targetUrl)
	failOnError(err)

	_, err = url.ParseRequestURI(targetUrl)
	failOnError(err)

	fmt.Print("Enter Requests: ")
	_, err = fmt.Scanln(&requests)
	failOnError(err)

	fmt.Print("Enter Concurrency: ")
	_, err = fmt.Scanln(&concurrency)
	failOnError(err)

	fmt.Print("Enter Delay: ")
	_, err = fmt.Scanln(&delay)
	failOnError(err)
}

func failOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
	return
}
