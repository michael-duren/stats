package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"ghstats/internal/cache"
	"ghstats/internal/github"
	"ghstats/internal/server"
)

func main() {
	tokens := loadTokens()
	if len(tokens) == 0 {
		log.Println("warning: no GitHub tokens set (GITHUB_TOKEN / PAT_1..); running unauthenticated at 60 req/hr")
	} else {
		log.Printf("loaded %d GitHub token(s)", len(tokens))
	}

	allowed := os.Getenv("ALLOWED_USERNAME")
	if allowed != "" {
		log.Printf("locked to a single user: %s", allowed)
	} else {
		log.Println("ALLOWED_USERNAME not set; serving any username")
	}

	gh := github.NewClient(tokens)
	c := cache.New()
	srv := server.New(gh, c, allowed)

	addr := ":" + envOr("PORT", "8080")
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("listening on %s", addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// loadTokens collects PATs from GITHUB_TOKEN and the PAT_1, PAT_2, ... series.
func loadTokens() []string {
	var tokens []string
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		tokens = append(tokens, t)
	}
	for i := 1; ; i++ {
		t := os.Getenv("PAT_" + itoa(i))
		if t == "" {
			break
		}
		tokens = append(tokens, t)
	}
	return tokens
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
