package observability

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func failure(w http.ResponseWriter, status int, msg string) {
	reply(w, status, map[string]string{"error": msg})
}
func filters(r *http.Request) (string, string, int64, int, error) {
	level := strings.ToLower(r.URL.Query().Get("level"))
	if level == "warning" {
		level = "warn"
	}
	if level != "" && level != "all" && level != "fatal" && level != "panic" {
		if _, err := parseLevel(level); err != nil {
			return "", "", 0, 0, err
		}
	}
	q := r.URL.Query().Get("q")
	if len(q) > 256 {
		return "", "", 0, 0, fmt.Errorf("query too long")
	}
	after := int64(0)
	limit := 500
	if v := r.URL.Query().Get("after_id"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			return "", "", 0, 0, fmt.Errorf("invalid cursor")
		}
		after = n
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 50000 {
			return "", "", 0, 0, fmt.Errorf("limit must be 1–50000")
		}
		limit = n
	}
	return level, q, after, limit, nil
}
func (s *Store) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value("user") == nil || r.Context().Value("role") != "admin" {
			failure(w, 401, "authentication required")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		switch r.URL.Path {
		case "/api/runtime/metrics":
			if r.Method != "GET" {
				failure(w, 405, "method not allowed")
				return
			}
			reply(w, 200, Metrics())
		case "/api/logs":
			switch r.Method {
			case "GET":
				level, q, after, limit, err := filters(r)
				if err != nil {
					failure(w, 400, err.Error())
					return
				}
				items, cursor, generation := s.snapshotState(level, q, after, limit)
				reply(w, 200, map[string]any{"items": items, "cursor": cursor, "generation": generation})
			case "DELETE":
				if err := s.Clear(); err != nil {
					failure(w, 500, "failed to persist cleared logs")
					return
				}
				reply(w, 200, map[string]bool{"cleared": true})
			default:
				failure(w, 405, "method not allowed")
			}
		case "/api/logs/settings":
			if r.Method == "PATCH" {
				var v Settings
				decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
				decoder.DisallowUnknownFields()
				if decoder.Decode(&v) != nil {
					failure(w, 400, "invalid settings")
					return
				}
				if err := s.UpdateSettings(v); err != nil {
					failure(w, 400, err.Error())
					return
				}
			} else if r.Method != "GET" {
				failure(w, 405, "method not allowed")
				return
			}
			reply(w, 200, s.Settings())
		case "/api/runtime/log-level":
			if r.Method == "PATCH" {
				var v struct {
					Level string `json:"level"`
				}
				decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
				decoder.DisallowUnknownFields()
				if decoder.Decode(&v) != nil {
					failure(w, 400, "invalid level")
					return
				}
				if err := s.SetRuntimeLevel(v.Level); err != nil {
					failure(w, 400, err.Error())
					return
				}
			} else if r.Method != "GET" {
				failure(w, 405, "method not allowed")
				return
			}
			reply(w, 200, map[string]string{"level": s.RuntimeLevel()})
		case "/api/events/logs":
			if r.Method != "GET" {
				failure(w, 405, "method not allowed")
				return
			}
			s.stream(w, r)
		default:
			failure(w, 404, "unknown endpoint")
		}
	})
}
func (s *Store) stream(w http.ResponseWriter, r *http.Request) {
	level, q, after, _, err := filters(r)
	if err != nil {
		failure(w, 400, err.Error())
		return
	}
	flush, ok := w.(http.Flusher)
	if !ok {
		failure(w, 500, "streaming unavailable")
		return
	}
	updates, unsubscribe, ok := s.Subscribe()
	if !ok {
		failure(w, 429, "too many log streams")
		return
	}
	defer unsubscribe()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	reset := s.ResetVersion()
	if supplied := r.URL.Query().Get("generation"); supplied != "" {
		v, e := strconv.ParseUint(supplied, 10, 64)
		if e != nil {
			failure(w, 400, "invalid generation")
			return
		}
		if v != reset {
			if _, err := fmt.Fprint(w, "event: log.reset\ndata: {}\n\n"); err != nil {
				return
			}
			flush.Flush()
			return
		}
	}
	send := func() bool {
		_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(15 * time.Second))
		items, cursor := s.Snapshot(level, q, after, 50000)
		for _, entry := range items {
			data, _ := json.Marshal(entry)
			if _, err := fmt.Fprintf(w, "id: %d\nevent: log.entry\ndata: %s\n\n", entry.ID, data); err != nil {
				return false
			}
		}
		after = cursor
		flush.Flush()
		return true
	}
	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return
	}
	if !send() {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.done:
			return
		case <-updates:
			if version := s.ResetVersion(); version != reset {
				reset = version
				if _, err := fmt.Fprint(w, "event: log.reset\ndata: {}\n\n"); err != nil {
					return
				}
			}
			if !send() {
				return
			}
		case <-heartbeat.C:
			_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(15 * time.Second))
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			flush.Flush()
		}
	}
}
