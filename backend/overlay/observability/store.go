package observability

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Entry struct {
	ID      int64             `json:"id"`
	TS      string            `json:"ts"`
	Level   string            `json:"level"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}
type Settings struct {
	MaxEntries int `json:"maxEntries"`
	MaxBytes   int `json:"maxBytes"`
}
type savedSettings struct {
	Settings
	RuntimeLevel string `json:"runtimeLevel,omitempty"`
}
type record struct {
	entry Entry
	data  []byte
}
type Store struct {
	mu          sync.Mutex
	diskMu      sync.Mutex
	records     []record
	head        int
	bytes       int
	sequence    int64
	revision    uint64
	reset       uint64
	persisted   uint64
	settings    Settings
	override    string
	logger      *logrus.Logger
	bound       map[*logrus.Logger]bool
	subscribers map[chan struct{}]struct{}
	directory   string
	done        chan struct{}
	wg          sync.WaitGroup
	closeOnce   sync.Once
}

var Default *Store
var sensitive = regexp.MustCompile(`(?i)(password|passwd|token|secret|authorization|credential|private.?key)`)
var link = regexp.MustCompile(`(?i)\b(?:ss|ssr|vmess|vless|trojan|socks5?|https?)://[^\s"'<>]+`)
var credential = regexp.MustCompile(`(?i)(?:Bearer\s+[A-Za-z0-9_.-]+|(?:github_pat_|ghp_)[A-Za-z0-9_]+)`)

func redact(value string) string {
	return credential.ReplaceAllString(link.ReplaceAllString(value, "[redacted-link]"), "[redacted]")
}
func validSettings(v Settings) bool {
	return v.MaxEntries >= 500 && v.MaxEntries <= 50000 && v.MaxBytes >= 5*1024*1024 && v.MaxBytes <= 200*1024*1024
}

func New(directory string) (*Store, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	if err := os.Chmod(directory, 0700); err != nil {
		return nil, err
	}
	s := &Store{directory: directory, settings: Settings{500, 5 * 1024 * 1024}, sequence: time.Now().UnixMilli() * 1000, bound: map[*logrus.Logger]bool{}, subscribers: map[chan struct{}]struct{}{}, done: make(chan struct{})}
	if data, err := os.ReadFile(filepath.Join(directory, "settings.json")); err == nil {
		var v savedSettings
		if json.Unmarshal(data, &v) == nil && validSettings(v.Settings) {
			s.settings = v.Settings
			if _, err := parseLevel(v.RuntimeLevel); err == nil {
				s.override = v.RuntimeLevel
			}
		}
	}
	if f, err := os.Open(filepath.Join(directory, "entries.jsonl")); err == nil {
		scan := bufio.NewScanner(f)
		scan.Buffer(make([]byte, 4096), 32*1024)
		for scan.Scan() {
			var entry Entry
			if json.Unmarshal(scan.Bytes(), &entry) == nil && entry.ID > 0 {
				data, _ := json.Marshal(entry)
				s.records = append(s.records, record{entry, data})
				s.bytes += len(data) + 1
				if entry.ID > s.sequence {
					s.sequence = entry.ID
				}
				s.trimLocked()
			}
		}
		f.Close()
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		timer := time.NewTicker(time.Second)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				_ = s.Flush()
			case <-s.done:
				return
			}
		}
	}()
	return s, nil
}
func Initialize(directory string) error {
	s, err := New(filepath.Join(directory, "logs", "console"))
	if err != nil {
		return err
	}
	Default = s
	Bind(logrus.StandardLogger())
	Activate(logrus.StandardLogger())
	return nil
}
func Bind(logger *logrus.Logger) {
	if Default == nil {
		return
	}
	s := Default
	s.mu.Lock()
	if !s.bound[logger] {
		logger.AddHook(s)
		s.bound[logger] = true
	}
	override := s.override
	s.mu.Unlock()
	if level, err := parseLevel(override); err == nil {
		logger.SetLevel(level)
	}
}
func Activate(logger *logrus.Logger) {
	if Default == nil {
		return
	}
	Default.mu.Lock()
	Default.logger = logger
	for old := range Default.bound {
		if old != logger && old != logrus.StandardLogger() {
			delete(Default.bound, old)
		}
	}
	Default.mu.Unlock()
}
func Discard(logger *logrus.Logger) {
	if Default == nil {
		return
	}
	Default.mu.Lock()
	if logger != Default.logger && logger != logrus.StandardLogger() {
		delete(Default.bound, logger)
	}
	Default.mu.Unlock()
}
func (s *Store) Levels() []logrus.Level { return logrus.AllLevels }
func (s *Store) Fire(entry *logrus.Entry) error {
	fields := map[string]string{}
	for k, v := range entry.Data {
		if sensitive.MatchString(k) {
			fields[k] = "[redacted]"
		} else {
			fields[k] = redact(fmt.Sprint(v))
			if len(fields[k]) > 2048 {
				fields[k] = fields[k][:2048] + "…"
			}
		}
	}
	msg := redact(entry.Message)
	if len(msg) > 8192 {
		msg = msg[:8192] + "…"
	}
	level := entry.Level.String()
	if level == "warning" {
		level = "warn"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	e := Entry{s.sequence, entry.Time.UTC().Format(time.RFC3339Nano), level, msg, fields}
	data, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	if len(data) > 24*1024 {
		e.Fields = map[string]string{"note": "oversized fields omitted"}
		data, _ = json.Marshal(e)
	}
	s.records = append(s.records, record{e, data})
	s.bytes += len(data) + 1
	s.revision++
	s.trimLocked()
	s.notifyLocked()
	return nil
}
func (s *Store) trimLocked() {
	for s.head < len(s.records) && (len(s.records)-s.head > s.settings.MaxEntries || s.bytes > s.settings.MaxBytes) {
		s.bytes -= len(s.records[s.head].data) + 1
		s.records[s.head] = record{}
		s.head++
	}
	if s.head >= 1024 && s.head*2 >= len(s.records) {
		s.records = append([]record(nil), s.records[s.head:]...)
		s.head = 0
	}
}
func (s *Store) notifyLocked() {
	for c := range s.subscribers {
		select {
		case c <- struct{}{}:
		default:
		}
	}
}
func matches(e Entry, level, q string) bool {
	if level != "" && level != "all" && e.Level != level {
		return false
	}
	if q == "" {
		return true
	}
	if strings.Contains(strings.ToLower(e.Message), q) {
		return true
	}
	for k, v := range e.Fields {
		if strings.Contains(strings.ToLower(k+" "+v), q) {
			return true
		}
	}
	return false
}
func (s *Store) Snapshot(level, q string, after int64, limit int) ([]Entry, int64) {
	items, cursor, _ := s.snapshotState(level, q, after, limit)
	return items, cursor
}
func (s *Store) snapshotState(level, q string, after int64, limit int) ([]Entry, int64, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []Entry{}
	q = strings.ToLower(strings.TrimSpace(q))
	for _, r := range s.records[s.head:] {
		if r.entry.ID > after && matches(r.entry, level, q) {
			items = append(items, r.entry)
		}
	}
	if len(items) > limit {
		items = items[len(items)-limit:]
	}
	return items, s.sequence, s.reset
}
func (s *Store) Subscribe() (chan struct{}, func(), bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.subscribers) >= 32 {
		return nil, nil, false
	}
	c := make(chan struct{}, 1)
	s.subscribers[c] = struct{}{}
	return c, func() { s.mu.Lock(); delete(s.subscribers, c); s.mu.Unlock() }, true
}
func atomicFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
func (s *Store) Flush() error {
	s.diskMu.Lock()
	defer s.diskMu.Unlock()
	s.mu.Lock()
	revision := s.revision
	if revision == s.persisted {
		s.mu.Unlock()
		return nil
	}
	data := make([]byte, 0, s.bytes)
	for _, r := range s.records[s.head:] {
		data = append(data, r.data...)
		data = append(data, '\n')
	}
	s.mu.Unlock()
	if err := atomicFile(filepath.Join(s.directory, "entries.jsonl"), data); err != nil {
		return err
	}
	s.mu.Lock()
	s.persisted = revision
	s.mu.Unlock()
	return nil
}
func (s *Store) Clear() error {
	s.diskMu.Lock()
	defer s.diskMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := atomicFile(filepath.Join(s.directory, "entries.jsonl"), nil); err != nil {
		return err
	}
	s.records = nil
	s.head = 0
	s.bytes = 0
	s.reset++
	s.revision++
	s.persisted = s.revision
	s.notifyLocked()
	return nil
}
func (s *Store) ResetVersion() uint64 { s.mu.Lock(); defer s.mu.Unlock(); return s.reset }
func (s *Store) Settings() Settings   { s.mu.Lock(); defer s.mu.Unlock(); return s.settings }
func (s *Store) UpdateSettings(v Settings) error {
	if !validSettings(v) {
		return fmt.Errorf("maxEntries must be 500–50000; maxBytes must be 5–200 MB")
	}
	s.diskMu.Lock()
	s.mu.Lock()
	data, _ := json.Marshal(savedSettings{v, s.override})
	err := atomicFile(filepath.Join(s.directory, "settings.json"), data)
	if err == nil {
		s.settings = v
		s.trimLocked()
		s.revision++
		s.notifyLocked()
	}
	s.mu.Unlock()
	s.diskMu.Unlock()
	if err != nil {
		return err
	}
	return s.Flush()
}
func parseLevel(level string) (logrus.Level, error) {
	switch level {
	case "error", "warn", "info", "debug", "trace":
		return logrus.ParseLevel(level)
	}
	return 0, fmt.Errorf("invalid log level")
}
func (s *Store) RuntimeLevel() string {
	s.mu.Lock()
	logger := s.logger
	s.mu.Unlock()
	if logger == nil {
		return "info"
	}
	v := logger.GetLevel().String()
	if v == "warning" {
		v = "warn"
	}
	return v
}
func (s *Store) SetRuntimeLevel(level string) error {
	v, err := parseLevel(level)
	if err != nil {
		return err
	}
	s.diskMu.Lock()
	s.mu.Lock()
	data, _ := json.Marshal(savedSettings{s.settings, level})
	err = atomicFile(filepath.Join(s.directory, "settings.json"), data)
	logger := s.logger
	if err == nil {
		s.override = level
	}
	s.mu.Unlock()
	s.diskMu.Unlock()
	if err != nil {
		return err
	}
	if logger != nil {
		logger.SetLevel(v)
	}
	logrus.SetLevel(v)
	logrus.WithField("level", level).Warn("Runtime log level updated")
	return nil
}
func (s *Store) Close() error {
	s.closeOnce.Do(func() { close(s.done); s.wg.Wait() })
	return s.Flush()
}
