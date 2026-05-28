// Package web serves the HTMX + Tailwind landing/playground UI that drives the
// SVG card endpoints under /api.
package web

import (
	"net/http"
	"net/url"
	"sort"
	"strings"

	"ghstats/internal/themes"
)

// CardType is one selectable card in the playground form.
type CardType struct {
	Value string
	Label string
}

var cardTypes = []CardType{
	{Value: "streak", Label: "Streak & Total Contributions"},
	{Value: "top-langs", Label: "Most Used Languages"},
	{Value: "stats", Label: "Stats"},
}

// endpoints maps a card type to its SVG API path.
var endpoints = map[string]string{
	"streak":    "/api/streak",
	"top-langs": "/api/top-langs",
	"stats":     "/api",
}

// IndexData is the model for the landing page.
type IndexData struct {
	Cards  []CardType
	Themes []string
}

// PreviewData is the model for the HTMX preview fragment.
type PreviewData struct {
	HasResult bool
	APIPath   string
	FullURL   string
	Markdown  string
}

// Index renders the playground landing page.
func Index(w http.ResponseWriter, r *http.Request) {
	data := IndexData{Cards: cardTypes, Themes: sortedThemes()}
	_ = IndexPage(data).Render(r.Context(), w)
}

// Preview renders the live card preview fragment from the form's query params.
func Preview(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	username := strings.TrimSpace(q.Get("username"))
	card := q.Get("card")
	theme := q.Get("theme")
	showIcons := q.Get("show_icons") == "on" || q.Get("show_icons") == "true"

	endpoint, ok := endpoints[card]
	if !ok {
		endpoint, card = endpoints["streak"], "streak"
	}

	var data PreviewData
	if username == "" {
		_ = PreviewFragment(data).Render(r.Context(), w)
		return
	}

	params := url.Values{}
	params.Set("username", username)
	if theme != "" && theme != "default" {
		params.Set("theme", theme)
	}
	if showIcons {
		params.Set("show_icons", "true")
	}
	apiPath := endpoint + "?" + params.Encode()
	base := baseURL(r)

	data = PreviewData{
		HasResult: true,
		APIPath:   apiPath,
		FullURL:   base + apiPath,
		Markdown:  "![" + card + "](" + base + apiPath + ")",
	}
	_ = PreviewFragment(data).Render(r.Context(), w)
}

func sortedThemes() []string {
	out := make([]string, 0, len(themes.Registry))
	for k := range themes.Registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
