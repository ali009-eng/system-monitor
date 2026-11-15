package main

import (
	"fmt"
	"net/http"
)
func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "you asked for:", r.URL.Path)
	fmt.Fprintln(w, "method:", r.Method)
}
func main() {
	http.HandleFunc("/status", hello)
	http.ListenAndServe(":8080", nil)
}
