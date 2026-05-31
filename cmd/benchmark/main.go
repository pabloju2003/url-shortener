package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	totalRequests = 1000
	concurrency   = 50
)

type result struct {
	duration time.Duration
	success  bool
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <code>\n", os.Args[0])
		os.Exit(1)
	}
	target := "http://localhost:8080/" + os.Args[1]

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}

	jobs := make(chan struct{}, totalRequests)
	for range totalRequests {
		jobs <- struct{}{}
	}
	close(jobs)

	results := make(chan result, totalRequests)

	var wg sync.WaitGroup
	start := time.Now()

	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				t0 := time.Now()
				resp, err := client.Get(target)
				dur := time.Since(t0)
				ok := err == nil && resp.StatusCode < 500
				if resp != nil {
					resp.Body.Close()
				}
				results <- result{duration: dur, success: ok}
			}
		}()
	}

	wg.Wait()
	totalTime := time.Since(start)
	close(results)

	var (
		sumDur       time.Duration
		minDur       = time.Duration(1<<63 - 1)
		maxDur       time.Duration
		successCount int
	)
	for r := range results {
		sumDur += r.duration
		if r.duration < minDur {
			minDur = r.duration
		}
		if r.duration > maxDur {
			maxDur = r.duration
		}
		if r.success {
			successCount++
		}
	}

	avgDur := sumDur / totalRequests
	reqPerSec := float64(totalRequests) / totalTime.Seconds()
	successRate := float64(successCount) / float64(totalRequests) * 100

	fmt.Printf("\n=== URL Shortener Benchmark ===\n")
	fmt.Printf("Requests:     %d\n", totalRequests)
	fmt.Printf("Concurrency:  %d\n", concurrency)
	fmt.Printf("Total time:   %.2fs\n", totalTime.Seconds())
	fmt.Printf("Req/sec:      %.0f\n", reqPerSec)
	fmt.Printf("Latency avg:  %dms\n", avgDur.Milliseconds())
	fmt.Printf("Latency min:  %dms\n", minDur.Milliseconds())
	fmt.Printf("Latency max:  %dms\n", maxDur.Milliseconds())
	fmt.Printf("Success rate: %.0f%%\n", successRate)
}
