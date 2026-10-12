package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func put(s *Store, level logrus.Level, msg string) {
	_ = s.Fire(&logrus.Entry{Time: time.Now(), Level: level, Message: msg, Data: logrus.Fields{"node": "demo", "password": "do-not-expose"}})
}
func TestCaptureFiltersRetentionAndRedaction(t *testing.T) {
	s := testStore(t)
	for i := 0; i < 700; i++ {
		put(s, logrus.InfoLevel, fmt.Sprintf("row-%d socks5://user:password@host:1080", i))
	}
	items, _ := s.Snapshot("all", "", 0, 50000)
	if len(items) != 500 {
		t.Fatal(len(items))
	}
	if strings.Contains(items[0].Message, "password") || items[0].Fields["password"] != "[redacted]" {
		t.Fatal("secret exposed")
	}
	items, _ = s.Snapshot("error", "", 0, 100)
	if len(items) != 0 {
		t.Fatal("wrong filter")
	}
	put(s, logrus.ErrorLevel, "unique-search")
	items, _ = s.Snapshot("error", "unique-search", 0, 500)
	if len(items) != 1 {
		t.Fatal("search missed")
	}
}
func TestConcurrentCaptureClearAndPersistence(t *testing.T) {
	s := testStore(t)
	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				put(s, logrus.WarnLevel, "parallel")
				_, _ = s.Snapshot("all", "", 0, 500)
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			_ = s.Clear()
			_ = s.Flush()
		}
	}()
	wg.Wait()
	_ = s.Clear()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(s.directory, "entries.jsonl"))
	if len(data) != 0 {
		t.Fatal("cleared records resurrected")
	}
}
func TestRestartSettingsAndByteLimit(t *testing.T) {
	s := testStore(t)
	if err := s.UpdateSettings(Settings{50000, 5 * 1024 * 1024}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2000; i++ {
		put(s, logrus.InfoLevel, strings.Repeat("x", 8000))
	}
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	stat, _ := os.Stat(filepath.Join(s.directory, "entries.jsonl"))
	if stat.Size() > 5*1024*1024 {
		t.Fatal("byte limit failed")
	}
	s.Close()
	next, err := New(s.directory)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	if next.Settings().MaxEntries != 50000 {
		t.Fatal("settings lost")
	}
	items, _ := next.Snapshot("all", "", 0, 50000)
	if len(items) == 0 || len(items) > 700 {
		t.Fatal(len(items))
	}
}
func authorized(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(context.WithValue(r.Context(), "user", true), "role", "admin"))
}
func TestAPIRequiresAuthenticationAndValidSettings(t *testing.T) {
	s := testStore(t)
	r := httptest.NewRequest("GET", "/api/logs", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
	r = authorized(httptest.NewRequest("PATCH", "/api/logs/settings", strings.NewReader(`{"maxEntries":1,"maxBytes":1}`)))
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}
func TestStreamDeliversLiveEventsAndReleasesDisconnectedClient(t *testing.T) {
	s := testStore(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { s.Handler().ServeHTTP(w, authorized(r)) }))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/events/logs?after_id=0", nil)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	put(s, logrus.InfoLevel, "live-positive-control")
	buffer := make([]byte, 4096)
	var got string
	for !strings.Contains(got, "live-positive-control") {
		n, err := response.Body.Read(buffer)
		if err != nil {
			t.Fatal(err)
		}
		got += string(buffer[:n])
	}
	cancel()
	response.Body.Close()
	for i := 0; i < 50; i++ {
		s.mu.Lock()
		count := len(s.subscribers)
		s.mu.Unlock()
		if count == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("subscriber leaked")
}
func TestBindDoesNotDuplicateHook(t *testing.T) {
	old := Default
	defer func() { Default = old }()
	s := testStore(t)
	Default = s
	log := logrus.New()
	log.Out = io.Discard
	Bind(log)
	Bind(log)
	Activate(log)
	log.Info("one entry")
	items, _ := s.Snapshot("all", "", 0, 500)
	if len(items) != 1 {
		t.Fatal("duplicate capture")
	}
	payload, _ := json.Marshal(items)
	if strings.Contains(string(payload), "do-not-expose") {
		t.Fatal("unexpected secret")
	}
}

func TestClearBetweenSnapshotAndConnectTriggersReset(t *testing.T) {
	s := testStore(t)
	put(s, logrus.InfoLevel, "old")
	_, _, generation := s.snapshotState("all", "", 0, 500)
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := authorized(httptest.NewRequest("GET", fmt.Sprintf("/api/events/logs?generation=%d", generation), nil))
	s.Handler().ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "event: log.reset") {
		t.Fatal("stale UI not reset")
	}
}
func TestRuntimeLevelPersistsAndFailedCandidateDoesNotBecomeActive(t *testing.T) {
	old := Default
	defer func() { Default = old }()
	s := testStore(t)
	Default = s
	active := logrus.New()
	active.Out = io.Discard
	Bind(active)
	Activate(active)
	if err := s.SetRuntimeLevel("debug"); err != nil {
		t.Fatal(err)
	}
	candidate := logrus.New()
	Bind(candidate)
	Discard(candidate)
	if err := s.SetRuntimeLevel("error"); err != nil {
		t.Fatal(err)
	}
	if active.GetLevel() != logrus.ErrorLevel {
		t.Fatal("active logger unchanged")
	}
	s.Close()
	next, err := New(s.directory)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	Default = next
	restarted := logrus.New()
	Bind(restarted)
	Activate(restarted)
	if restarted.GetLevel() != logrus.ErrorLevel {
		t.Fatal("runtime level lost on restart")
	}
}
