package watcher

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	resultCh := make(chan *Result, 100)        // 결과 수집용 채널
	jobCh := make(chan int, w.config.Requests) // 작업 큐
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

	// 작업 큐에 모든 요청 추가
	for i := 0; i < w.config.Requests; i++ {
		jobCh <- i
	}
	close(jobCh)

	// Concurrency 개수만큼 워커 시작
	for i := 0; i < w.config.Concurrency; i++ {
		wg.Add(1)
		go w.worker(jobCh, resultCh, &wg)
	}

	// 모든 요청 완료 대기
	wg.Wait()
	close(resultCh)

	// 수집 완료 대기
	collectWg.Wait()

	// 최종 통계 출력
	w.totalPrint(results)

	// Markdown 리포트 생성
	if err := w.generateMarkdownReport(results); err != nil {
		fmt.Printf("리포트 생성 실패: %v\n", err)
	} else if w.config.ReportFile != "" {
		fmt.Printf("리포트 파일: %s\n", w.config.ReportFile)
	}
}

func (w *Watcher) worker(jobCh <-chan int, resultCh chan *Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for range jobCh {
		// 요청 실행 (스크립트 변수 전달)
		result := MeasureRequestTime(w.config, w.scriptExecutor.GetVars())

		// 콘솔 출력 (응답 즉시)
		w.print(result)

		// 채널로 전송 (파일 쓰기 + 통계용)
		resultCh <- result

		// 각 워커가 독립적으로 delay 적용 (응답 시간 차이로 인해 묶음이 아님)
		if w.delay > 0 {
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

// generateMarkdownReport Markdown 테이블 형식의 리포트를 생성합니다
func (w *Watcher) generateMarkdownReport(results []*Result) error {
	if w.config.ReportFile == "" {
		return nil
	}

	file, err := os.Create(w.config.ReportFile)
	if err != nil {
		return fmt.Errorf("리포트 파일 생성 실패: %w", err)
	}
	defer file.Close()

	// 제목과 요약 정보
	var sb strings.Builder
	sb.WriteString("# 요청 결과 리포트\n\n")
	sb.WriteString(fmt.Sprintf("생성 시간: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 테이블 헤더
	sb.WriteString("| # | 타임스탬프 | 메서드 | URL | 상태 코드 | 소요 시간 (ms) | 컨텐츠 길이 | 결과 |\n")
	sb.WriteString("|---|-----------|--------|-----|----------|--------------|-------------|------|\n")

	// 각 결과를 테이블 행으로 추가
	for i, result := range results {
		timestamp := result.Timestamp.Format("15:04:05.000")
		status := fmt.Sprintf("%d", result.StatusCode)
		elapsed := fmt.Sprintf("%d", result.ElapsedMs)
		contentLength := fmt.Sprintf("%d", result.ContentLength)

		// 결과 상태 (성공/에러)
		resultStatus := "✅ 성공"
		if result.Error != nil {
			resultStatus = fmt.Sprintf("❌ 에러: %s", result.ErrorMessage)
			status = "-"
			if result.StatusCode == 0 {
				status = "-"
			}
		}

		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s | %s | %s |\n",
			i+1,
			timestamp,
			result.Method,
			result.URL,
			status,
			elapsed,
			contentLength,
			resultStatus,
		))
	}

	// 통계 요약
	sb.WriteString("\n## 통계 요약\n\n")

	totalRequests := len(results)
	totalErrors := 0
	totalDuration := time.Duration(0)
	var minDuration time.Duration
	var maxDuration time.Duration
	successCount := 0

	for _, result := range results {
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

	sb.WriteString(fmt.Sprintf("- **전체 요청**: %d\n", totalRequests))
	sb.WriteString(fmt.Sprintf("- **성공**: %d\n", successCount))
	sb.WriteString(fmt.Sprintf("- **에러**: %d\n", totalErrors))

	if successCount > 0 {
		sb.WriteString(fmt.Sprintf("\n### 응답 시간\n\n"))
		sb.WriteString(fmt.Sprintf("- **평균**: %dms\n", avgDuration.Milliseconds()))
		sb.WriteString(fmt.Sprintf("- **최소**: %dms\n", minDuration.Milliseconds()))
		sb.WriteString(fmt.Sprintf("- **최대**: %dms\n", maxDuration.Milliseconds()))
		sb.WriteString(fmt.Sprintf("- **합계**: %dms\n", totalDuration.Milliseconds()))
	}

	// 파일에 쓰기
	if _, err := file.WriteString(sb.String()); err != nil {
		return fmt.Errorf("파일 쓰기 실패: %w", err)
	}

	return nil
}
