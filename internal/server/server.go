package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"

	"ghstats/internal/cache"
	"ghstats/internal/cards"
	"ghstats/internal/github"
	"ghstats/internal/web"
)

const (
	minCacheSeconds     = 1800   // 30m floor to protect the rate limit
	maxCacheSeconds     = 86400  // 24h ceiling
	defaultCacheSeconds = 14400  // 4h
	errorCacheSeconds   = 10
)

// Server wires the GitHub client and cache into HTTP handlers.
type Server struct {
	gh    *github.Client
	cache *cache.TTL
}

func New(gh *github.Client, c *cache.TTL) *Server {
	return &Server{gh: gh, cache: c}
}

// Routes returns the configured HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api", s.handleStats)
	mux.HandleFunc("GET /api/top-langs", s.handleTopLangs)
	mux.HandleFunc("GET /api/streak", s.handleStreak)
	mux.HandleFunc("GET /api/pin", s.handlePin)
	mux.HandleFunc("GET /api/gist", s.handleGist)
	mux.HandleFunc("GET /{$}", web.Index)
	mux.HandleFunc("GET /preview", web.Preview)
	return mux
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	username := strings.TrimSpace(firstParam(q, "username", "u"))
	if username == "" {
		s.writeError(w, r, "missing ?username", errorCacheSeconds)
		return
	}

	opts := parseOptions(q)
	includeAll := parseBool(q.Get("include_all_commits"), false)
	countPrivate := parseBool(q.Get("count_private"), false)
	showRank := !parseBool(q.Get("hide_rank"), false)

	s.serve(w, r, func(ctx context.Context) (templ.Component, error) {
		stats, err := s.gh.FetchStats(ctx, username, includeAll, countPrivate)
		if err != nil {
			return nil, err
		}
		view := cards.BuildStatsView(stats, opts, time.Now().Year(), showRank, includeAll)
		return cards.StatsCard(view), nil
	})
}

func (s *Server) handleTopLangs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	username := strings.TrimSpace(firstParam(q, "username", "u"))
	if username == "" {
		s.writeError(w, r, "missing ?username", errorCacheSeconds)
		return
	}
	opts := parseOptions(q)
	limit := parseInt(q.Get("langs_count"), 5)

	s.serve(w, r, func(ctx context.Context) (templ.Component, error) {
		langs, err := s.gh.FetchTopLanguages(ctx, username)
		if err != nil {
			return nil, err
		}
		view := cards.BuildLangsView(langs, opts, limit)
		return cards.LangsCard(view), nil
	})
}

func (s *Server) handleStreak(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	username := strings.TrimSpace(firstParam(q, "username", "u"))
	if username == "" {
		s.writeError(w, r, "missing ?username", errorCacheSeconds)
		return
	}
	opts := parseOptions(q)

	s.serve(w, r, func(ctx context.Context) (templ.Component, error) {
		st, err := s.gh.FetchStreak(ctx, username)
		if err != nil {
			return nil, err
		}
		return cards.StreakCard(cards.BuildStreakView(st, opts)), nil
	})
}

func (s *Server) handlePin(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	username := strings.TrimSpace(firstParam(q, "username", "u"))
	repo := strings.TrimSpace(q.Get("repo"))
	if username == "" || repo == "" {
		s.writeError(w, r, "missing ?username and ?repo", errorCacheSeconds)
		return
	}
	opts := parseOptions(q)

	s.serve(w, r, func(ctx context.Context) (templ.Component, error) {
		rp, err := s.gh.FetchRepo(ctx, username, repo)
		if err != nil {
			return nil, err
		}
		view := cards.BuildRepoView(rp, opts)
		return cards.RepoCard(view), nil
	})
}

func (s *Server) handleGist(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := strings.TrimSpace(q.Get("id"))
	if id == "" {
		s.writeError(w, r, "missing ?id", errorCacheSeconds)
		return
	}
	opts := parseOptions(q)

	s.serve(w, r, func(ctx context.Context) (templ.Component, error) {
		g, err := s.gh.FetchGist(ctx, id)
		if err != nil {
			return nil, err
		}
		view := cards.BuildGistView(g, opts)
		return cards.RepoCard(view), nil
	})
}

// serve handles caching, rendering, header-setting, and error mapping that is
// identical across every card endpoint.
func (s *Server) serve(w http.ResponseWriter, r *http.Request, build func(context.Context) (templ.Component, error)) {
	cacheSeconds := clampCache(parseInt(r.URL.Query().Get("cache_seconds"), defaultCacheSeconds))
	key := r.URL.RequestURI()

	if cached, ok := s.cache.Get(key); ok {
		writeSVG(w, cached, cacheSeconds)
		return
	}

	component, err := build(r.Context())
	if err != nil {
		log.Printf("%s: %v", r.URL.RequestURI(), err)
		s.writeError(w, r, errorMessage(err), errorCacheSeconds)
		return
	}

	var buf bytes.Buffer
	if err := component.Render(r.Context(), &buf); err != nil {
		s.writeError(w, r, "render failed", errorCacheSeconds)
		return
	}
	out := buf.Bytes()
	s.cache.Set(key, out, time.Duration(cacheSeconds)*time.Second)
	writeSVG(w, out, cacheSeconds)
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, message string, cacheSeconds int) {
	var buf bytes.Buffer
	_ = cards.ErrorCard(message).Render(r.Context(), &buf)
	writeSVG(w, buf.Bytes(), cacheSeconds)
}

// errorMessage maps an upstream failure to a short, actionable card message,
// surfacing GitHub's own wording for auth/rate-limit problems.
func errorMessage(err error) string {
	var apiErr *github.APIError
	if !errors.As(err, &apiErr) {
		return "something went wrong"
	}
	lower := strings.ToLower(apiErr.Message)
	switch {
	case apiErr.Status == http.StatusNotFound:
		return "not found — check the username/repo/id"
	case apiErr.Status == http.StatusUnauthorized:
		return "github: bad credentials — set a valid GITHUB_TOKEN"
	case strings.Contains(lower, "rate limit"):
		return "github rate limit hit — set GITHUB_TOKEN or wait"
	case apiErr.Status == http.StatusForbidden:
		return "github forbidden (403) — set GITHUB_TOKEN; you may be rate limited"
	default:
		return fmt.Sprintf("github api error (%d)", apiErr.Status)
	}
}

func writeSVG(w http.ResponseWriter, body []byte, cacheSeconds int) {
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control",
		fmt.Sprintf("max-age=%d, s-maxage=%d, stale-while-revalidate=%d", cacheSeconds, cacheSeconds, cacheSeconds))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// --- query parsing helpers ---

func parseOptions(q map[string][]string) cards.Options {
	get := func(k string) string {
		if v, ok := q[k]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}
	opts := cards.ResolveOptions(
		get("theme"),
		get("title_color"),
		get("icon_color"),
		get("text_color"),
		get("bg_color"),
		get("border_color"),
	)
	opts.HideBorder = parseBool(get("hide_border"), false)
	opts.HideTitle = parseBool(get("hide_title"), false)
	opts.ShowIcons = parseBool(get("show_icons"), false)
	opts.CustomTitle = get("custom_title")
	opts.CardWidth = parseInt(get("card_width"), 0)
	for _, h := range strings.Split(get("hide"), ",") {
		h = strings.TrimSpace(strings.ToLower(h))
		if h != "" {
			opts.Hide[h] = true
		}
	}
	return opts
}

func firstParam(q map[string][]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := q[k]; ok && len(v) > 0 && v[0] != "" {
			return v[0]
		}
	}
	return ""
}

func parseBool(s string, def bool) bool {
	if s == "" {
		return def
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return b
}

func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func clampCache(n int) int {
	if n < minCacheSeconds {
		return minCacheSeconds
	}
	if n > maxCacheSeconds {
		return maxCacheSeconds
	}
	return n
}
