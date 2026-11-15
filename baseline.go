package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)
var webhook = os.Getenv("DISCORD_WEBHOOK")
var client = &http.Client{Timeout: 5 * time.Second}

func statusHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`SELECT DISTINCT ON (url) url, status, duration_ms, checked_at
			FROM results ORDER BY url, checked_at DESC`)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var url string
			var status, ms int
			var t time.Time
			if err := rows.Scan(&url, &status, &ms, &t); err != nil {
				fmt.Println("scan:", err)
				continue
			}
			fmt.Fprintf(w, "%s  %d  %dms  %s\n", url, status, ms, t.Format(time.RFC3339))
		}
	}
}

func saveResult(db *sql.DB, res Result) error {
	_, err := db.Exec(
		"INSERT INTO results (url, status, duration_ms) VALUES ($1, $2, $3)",
		res.URL, res.Status, res.Duration.Milliseconds(),
	)
	return err
}

func notify(webhook, msg string) error {
	if webhook == "" {
		return nil
	}

	payload := fmt.Sprintf(`{"content": %q}`, msg)
	resp, err := client.Post(webhook, "application/json", strings.NewReader(payload))
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
	fails := make(map[string]int) // url -> consecutive failures
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/status", statusHandler(db))
	go http.ListenAndServe(":8080", nil)

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

			if err := saveResult(db, res); err != nil {
				fmt.Println("Failed to save result:", err)
			}
			if res.IsUp() {
				fails[res.URL] = 0
			} else {
				fails[res.URL]++
			}

			prev, seen := last[res.URL]
			now := fails[res.URL] < 3

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
