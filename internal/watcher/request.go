package watcher

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Result struct {
	Timestamp    time.Time     `json:"timestamp"`
	Method       string        `json:"method"`
	URL          string        `json:"url"`
	StatusCode   int           `json:"status_code"`
	Elapsed      time.Duration `json:"elapsed_ns"`
	ElapsedMs    int64         `json:"elapsed_ms"`
	Error        error         `json:"error,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
	ResponseBody string        `json:"response_body,omitempty"`
}

func MeasureRequestTime(url, method string, saveBody bool) *Result {
	timestamp := time.Now()
	result := &Result{
		Timestamp: timestamp,
		Method:    strings.ToUpper(method),
		URL:       url,
	}

	req, err := http.NewRequest(result.Method, url, nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		result.ErrorMessage = result.Error.Error()
		return result
	}

	start := time.Now()
	client := &http.Client{}
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	result.Elapsed = elapsed
	result.ElapsedMs = elapsed.Milliseconds()

	if err != nil {
		result.Error = err
		result.ErrorMessage = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// 응답 본문 저장 (옵션)
	if saveBody {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			result.ResponseBody = string(body)
		}
	}

	return result
}
