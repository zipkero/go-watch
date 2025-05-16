package watcher

import (
	"fmt"
	"sync"
	"time"
)

type Watcher struct {
	URL         string
	Requests    int
	Concurrency int
	delay       time.Duration
}

func NewWatcher(url string, requests, concurrency int, delay time.Duration) *Watcher {
	return &Watcher{
		URL:         url,
		Requests:    requests,
		Concurrency: concurrency,
		delay:       delay,
	}
}

func (w *Watcher) Start() {
	var results []*Result
	completeCh := make(chan []*Result)
	var wg sync.WaitGroup
	avgRequests := w.Requests / w.Concurrency
	remainder := w.Requests % w.Concurrency
	for i := 0; i < w.Concurrency; i++ {
		assignCount := avgRequests
		if i < remainder {
			assignCount++
		}
		wg.Add(1)
		go w.startPool(completeCh, assignCount, &wg)
	}

	go func() {
		wg.Wait()
		close(completeCh)
	}()

	for i := 0; i < w.Concurrency; i++ {
		if result, ok := <-completeCh; ok {
			results = append(results, result...)
		}
	}

	w.totalPrint(results)
}

func (w *Watcher) startPool(completeCh chan []*Result, assignCount int, wg *sync.WaitGroup) {
	defer wg.Done()

	var workerResults []*Result
	var workerWg sync.WaitGroup

	workerCh := make(chan *Result)
	workerWg.Add(assignCount)

	go func() {
		for i := 0; i < assignCount; i++ {
			go func() {
				workerCh <- MeasureRequestTime(w.URL)
			}()

			time.Sleep(w.delay * time.Second)
		}
	}()

	go func() {
		for i := 0; i < assignCount; i++ {
			result := <-workerCh
			w.print(result)
			workerResults = append(workerResults, result)
			workerWg.Done()
		}
	}()

	workerWg.Wait()
	close(workerCh)

	completeCh <- workerResults
}

func (w *Watcher) print(result *Result) {
	fmt.Println("Status Code: ", result.StatusCode, " Elapsed: ", result.Elapsed, " Error: ", result.Error, "")
}

func (w *Watcher) totalPrint(results []*Result) {
	totalRequests := 0
	totalSuccess := 0
	totalErrors := 0
	totalDuration := time.Duration(0)
	avgDuration := time.Duration(0)
	minDuration := time.Duration(int(^uint(0) >> 1))
	maxDuration := time.Duration(0)
	for _, result := range results {
		totalRequests++
		if result.Error != nil {
			totalErrors++
		} else {
			totalDuration += result.Elapsed
			if minDuration > result.Elapsed {
				minDuration = result.Elapsed
			}
			if maxDuration < result.Elapsed {
				maxDuration = result.Elapsed
			}
			totalSuccess++
		}
	}

	avgDuration = totalDuration / time.Duration(totalSuccess)

	fmt.Println("==================================================")

	fmt.Println("Total Requests: ", totalRequests)
	fmt.Println("Total Success: ", totalSuccess)
	fmt.Println("Total Errors: ", totalErrors)
	fmt.Println("Total Duration: ", totalDuration)
	fmt.Println("Average Duration: ", avgDuration)
	fmt.Println("Min Duration: ", minDuration)
	fmt.Println("Max Duration: ", maxDuration)
}
