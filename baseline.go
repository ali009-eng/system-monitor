package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var webhook = os.Getenv("DISCORD_WEBHOOK")
var client = &http.Client{Timeout: 5 * time.Second}

func notify(webhook, msg string) error {
	if webhook == "" {
		return nil
	}

	payload := fmt.Sprintf(`{"content": %q}`, msg)
	resp, err := http.Post(webhook, "application/json", strings.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord returned status %d", resp.StatusCode)
	}

	return nil
}

type Result struct {
	URL      string
	Status   int
	Duration time.Duration
}

func (r Result) IsUp() bool {
	return r.Status >= 200 && r.Status <= 299
}

func runCheck(url string) Result {
	start := time.Now()
	status, _ := check(url)
	elapsed := time.Since(start)

	return Result{URL: url, Status: status, Duration: elapsed}
}

func check(url string) (int, error) {
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func main() {
	urls := []string{
		"https://google.com",
		"https://example.com",
		"https://github.com",
		"https://httpbin.org/status/404",
		"https://httpbin.org/status/500",
		"https://thisdoesnotexist.invalid",
		"http://localhost:8000",
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	last := make(map[string]bool) // url -> was up last round

	for range ticker.C {
		results := make([]Result, len(urls))
		var wg sync.WaitGroup

		for i, url := range urls {
			wg.Add(1)
			go func(i int, u string) {
				defer wg.Done()
				results[i] = runCheck(u)
			}(i, url)
		}
		wg.Wait()

		for _, res := range results {
			prev, seen := last[res.URL]
			now := res.IsUp()

			if !seen {
				fmt.Println("INITIAL:", res.URL, now)
			} else if prev != now {
				if now {
					msg := fmt.Sprintf("RECOVERED: %s (Status %d)", res.URL, res.Status)
					fmt.Println(msg)
					if err := notify(webhook, msg); err != nil {
						fmt.Println("Failed to send Discord alert:", err)
					}
				} else {
					msg := fmt.Sprintf("WENT DOWN: %s (Status %d)", res.URL, res.Status)
					fmt.Println(msg)
					if err := notify(webhook, msg); err != nil {
						fmt.Println("Failed to send Discord alert:", err)
					}
				}
			}

			last[res.URL] = now
		}
	}
}
