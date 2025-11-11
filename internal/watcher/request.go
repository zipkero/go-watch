package watcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zipkero/go-watch/internal/config"
	"github.com/zipkero/go-watch/internal/script"
)

type Result struct {
	Timestamp     time.Time     `json:"timestamp"`
	Method        string        `json:"method"`
	URL           string        `json:"url"`
	StatusCode    int           `json:"status_code"`
	Elapsed       time.Duration `json:"elapsed_ns"`
	ElapsedMs     int64         `json:"elapsed_ms"`
	Error         error         `json:"error,omitempty"`
	ErrorMessage  string        `json:"error_message,omitempty"`
	ResponseBody  string        `json:"response_body,omitempty"`
	ContentLength int64         `json:"content_length,omitempty"`
}

// MeasureRequestTime HTTP 요청 후 결과값 반환
func MeasureRequestTime(cfg *config.Config, vars map[string]interface{}) *Result {
	timestamp := time.Now()
	result := &Result{
		Timestamp: timestamp,
		Method:    strings.ToUpper(cfg.Method),
		URL:       cfg.URL,
	}

	// 템플릿 변수 치환
	requestURL := script.ReplaceTemplates(cfg.URL, vars)

	// URL에 QueryParams 추가
	if len(cfg.QueryParams) > 0 {
		parsedURL, err := url.Parse(requestURL)
		if err != nil {
			result.Error = fmt.Errorf("failed to parse URL: %w", err)
			result.ErrorMessage = result.Error.Error()
			return result
		}

		query := parsedURL.Query()
		replacedParams := script.ReplaceTemplatesInMap(cfg.QueryParams, vars)
		for key, value := range replacedParams {
			query.Set(key, value)
		}
		parsedURL.RawQuery = query.Encode()
		requestURL = parsedURL.String()
		result.URL = requestURL
	}

	// Body 준비
	var bodyReader io.Reader
	if cfg.Body != nil {
		bodyBytes, contentType, err := prepareBody(cfg.BodyType, cfg.Body)
		if err != nil {
			result.Error = fmt.Errorf("failed to prepare body: %w", err)
			result.ErrorMessage = result.Error.Error()
			return result
		}
		bodyReader = bytes.NewReader(bodyBytes)

		// Content-Type이 Headers에 없으면 자동 설정
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		if _, exists := cfg.Headers["Content-Type"]; !exists && contentType != "" {
			cfg.Headers["Content-Type"] = contentType
		}
	}

	// HTTP 요청 생성
	req, err := http.NewRequest(result.Method, requestURL, bodyReader)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		result.ErrorMessage = result.Error.Error()
		return result
	}

	// Headers 설정 (템플릿 변수 치환)
	replacedHeaders := script.ReplaceTemplatesInMap(cfg.Headers, vars)
	for key, value := range replacedHeaders {
		req.Header.Set(key, value)
	}

	// 요청 실행
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

	body, err := io.ReadAll(resp.Body)
	if err == nil {
		result.ContentLength = int64(len(body))
		if cfg.SaveResponseBody {
			result.ResponseBody = string(body)
		}
	}

	return result
}

// prepareBody body_type 에 따라 Content-Type을 반환
func prepareBody(bodyType string, body interface{}) ([]byte, string, error) {
	if body == nil {
		return nil, "", nil
	}

	switch strings.ToLower(bodyType) {
	case "json":
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return bodyBytes, "application/json", nil

	case "form":
		bodyMap, ok := body.(map[string]interface{})
		if !ok {
			return nil, "", fmt.Errorf("form body must be a map")
		}

		formData := url.Values{}
		for key, value := range bodyMap {
			formData.Set(key, fmt.Sprintf("%v", value))
		}
		return []byte(formData.Encode()), "application/x-www-form-urlencoded", nil

	case "xml":
		bodyStr, ok := body.(string)
		if !ok {
			return nil, "", fmt.Errorf("xml body must be a string")
		}
		return []byte(bodyStr), "application/xml", nil

	case "raw", "":
		bodyStr, ok := body.(string)
		if !ok {
			return nil, "", fmt.Errorf("raw body must be a string")
		}
		return []byte(bodyStr), "text/plain", nil

	default:
		return nil, "", fmt.Errorf("unsupported body type: %s", bodyType)
	}
}
