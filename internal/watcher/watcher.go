package watcher

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/zipkero/go-watch/internal/config"
	"github.com/zipkero/go-watch/internal/script"
)

type Watcher struct {
	config         *config.Config
	delay          time.Duration
	scriptExecutor *script.ScriptExecutor
}

func NewWatcher(cfg *config.Config) *Watcher {
	executor := script.NewScriptExecutor()

	// Pre-request 스크립트 실행
	if cfg.PreRequestScript != "" {
		if err := executor.Execute(cfg.PreRequestScript); err != nil {
			fmt.Printf("스크립트 실행 오류: %v\n", err)
		}
	}

	return &Watcher{
		config:         cfg,
		delay:          time.Duration(cfg.Delay),
		scriptExecutor: executor,
	}
}

func (w *Watcher) Start() {
	var results []*Result
	var resultsMutex sync.Mutex
	resultCh := make(chan *Result, 100) // 결과 수집용 채널
	var wg sync.WaitGroup

	// 결과 수집 및 파일 쓰기용 고루틴
	var collectWg sync.WaitGroup
	collectWg.Add(1)

	var file *os.File
	var encoder *json.Encoder
	var fileErr error

	// 파일 열기 (필요한 경우)
	if w.config.OutputFile != "" {
		file, fileErr = os.Create(w.config.OutputFile)
		if fileErr != nil {
			fmt.Printf("파일 생성 실패: %v\n", fileErr)
		} else {
			defer file.Close()
			encoder = json.NewEncoder(file)
		}
	}

	go func() {
		defer collectWg.Done()
		for result := range resultCh {
			resultsMutex.Lock()
			results = append(results, result)
			resultsMutex.Unlock()

			if encoder != nil {
				if err := encoder.Encode(result); err != nil {
					fmt.Printf("파일 쓰기 실패: %v\n", err)
				}
			}
		}
	}()

	// 요청 분배
	avgRequests := w.config.Requests / w.config.Concurrency
	remainder := w.config.Requests % w.config.Concurrency

	for i := 0; i < w.config.Concurrency; i++ {
		assignCount := avgRequests
		if i < remainder {
			assignCount++
		}
		wg.Add(1)
		go w.startPool(resultCh, assignCount, &wg)
	}

	// 모든 요청 완료 대기
	wg.Wait()
	close(resultCh)

	// 수집 완료 대기
	collectWg.Wait()

	// 최종 통계 출력
	w.totalPrint(results)
}

func (w *Watcher) startPool(resultCh chan *Result, assignCount int, wg *sync.WaitGroup) {
	defer wg.Done()

	for i := 0; i < assignCount; i++ {
		// 요청 실행 (스크립트 변수 전달)
		result := MeasureRequestTime(w.config, w.scriptExecutor.GetVars())

		// 콘솔 출력
		w.print(result)

		// 채널로 전송 (파일 쓰기 + 통계용)
		resultCh <- result

		// 다음 요청까지 대기
		if i < assignCount-1 {
			time.Sleep(w.delay * time.Second)
		}
	}
}

func (w *Watcher) print(result *Result) {
	errorMsg := ""
	if result.Error != nil {
		errorMsg = fmt.Sprintf(" Error: %v", result.Error)
	}

	fmt.Printf("Status: %d  Elapsed: %dms%s\n",
		result.StatusCode,
		result.ElapsedMs,
		errorMsg)
}

func (w *Watcher) totalPrint(results []*Result) {
	totalRequests := 0
	totalErrors := 0
	totalDuration := time.Duration(0)
	var minDuration time.Duration
	var maxDuration time.Duration
	successCount := 0

	for _, result := range results {
		totalRequests++

		if result.Error != nil {
			totalErrors++
		} else {
			totalDuration += result.Elapsed
			if successCount == 0 {
				minDuration = result.Elapsed
				maxDuration = result.Elapsed
			} else {
				if result.Elapsed < minDuration {
					minDuration = result.Elapsed
				}
				if result.Elapsed > maxDuration {
					maxDuration = result.Elapsed
				}
			}
			successCount++
		}
	}

	var avgDuration time.Duration
	if successCount > 0 {
		avgDuration = totalDuration / time.Duration(successCount)
	}

	fmt.Println("\n==================================================")
	fmt.Println("요청 통계")
	fmt.Println("==================================================")
	fmt.Printf("전체 요청: %d\n", totalRequests)
	fmt.Printf("성공: %d\n", successCount)
	fmt.Printf("에러: %d\n", totalErrors)

	fmt.Println("\n응답 시간")
	fmt.Println("==================================================")
	if successCount > 0 {
		fmt.Printf("평균: %dms\n", avgDuration.Milliseconds())
		fmt.Printf("최소: %dms\n", minDuration.Milliseconds())
		fmt.Printf("최대: %dms\n", maxDuration.Milliseconds())
		fmt.Printf("합계: %dms\n", totalDuration.Milliseconds())
	} else {
		fmt.Println("성공한 요청이 없습니다.")
	}

	if w.config.OutputFile != "" {
		fmt.Printf("\n결과 파일: %s\n", w.config.OutputFile)
	}
	fmt.Println("==================================================")
}
