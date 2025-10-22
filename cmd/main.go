package main

import (
	"fmt"
	"log"
	"os"

	"github.com/zipkero/go-watch/internal/config"
	"github.com/zipkero/go-watch/internal/watcher"
)

func main() {
	configPath := getConfigPath()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("설정 파일 로드 실패: %v", err)
	}

	fmt.Printf("설정 로드 완료: %s\n", configPath)
	fmt.Printf("URL: %s\n", cfg.URL)
	fmt.Printf("Method: %s\n", cfg.Method)
	fmt.Printf("Requests: %d\n", cfg.Requests)
	fmt.Printf("Concurrency: %d\n", cfg.Concurrency)
	fmt.Printf("Delay: %d초\n", cfg.Delay)

	if cfg.OutputFile != "" {
		fmt.Printf("출력 파일: %s\n", cfg.OutputFile)
	}
	fmt.Println()

	wc := watcher.NewWatcher(cfg)
	if wc == nil {
		log.Fatal("Watcher 생성 실패")
	}

	wc.Start()
}

func getConfigPath() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}

	defaultPath := "config.yaml"
	fmt.Printf("설정 파일 경로를 입력하세요 (기본값: %s): ", defaultPath)

	var path string
	fmt.Scanln(&path)

	if path == "" {
		return defaultPath
	}

	return path
}
