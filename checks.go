package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type Check struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

func listChecks(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, url FROM checks ORDER BY id")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		checks := []Check{} // empty slice (not nil) so it encodes as [] instead of null
		for rows.Next() {
			var c Check
			if err := rows.Scan(&c.ID, &c.URL); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			checks = append(checks, c)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(checks)
	}
}
