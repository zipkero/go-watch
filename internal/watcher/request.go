package watcher

import (
	"net/http"
	"time"
)

type Result struct {
	StatusCode int
	Elapsed    time.Duration
	Error      error
}

func MeasureRequestTime(url string) *Result {
	start := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		return &Result{Error: err}
	}
	defer resp.Body.Close()

	return &Result{StatusCode: resp.StatusCode, Elapsed: time.Since(start)}
}
