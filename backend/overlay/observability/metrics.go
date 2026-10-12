package observability

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var processStarted = time.Now()
var cpuMu sync.Mutex
var previousTicks float64
var previousSample time.Time
var lastCPU *float64

func Metrics() map[string]any {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	var rss any
	if data, err := os.ReadFile("/proc/self/status"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) > 1 {
					if n, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
						rss = n * 1024
					}
				}
			}
		}
	}
	cpuMu.Lock()
	defer cpuMu.Unlock()
	if data, err := os.ReadFile("/proc/self/stat"); err == nil {
		tail := strings.LastIndex(string(data), ")")
		if tail >= 0 {
			fields := strings.Fields(string(data)[tail+1:])
			if len(fields) > 12 {
				user, e1 := strconv.ParseFloat(fields[11], 64)
				system, e2 := strconv.ParseFloat(fields[12], 64)
				if e1 == nil && e2 == nil {
					ticks := user + system
					now := time.Now()
					if !previousSample.IsZero() && now.Sub(previousSample) >= time.Second {
						value := (ticks - previousTicks) / 100 / now.Sub(previousSample).Seconds() * 100
						if value < 0 {
							value = 0
						}
						lastCPU = &value
						previousTicks = ticks
						previousSample = now
					} else if previousSample.IsZero() {
						previousTicks = ticks
						previousSample = now
					}
				}
			}
		}
	}
	return map[string]any{"rssBytes": rss, "heapBytes": memory.HeapAlloc, "goroutines": runtime.NumGoroutine(), "cpuPercent": lastCPU, "processUptimeSeconds": int64(time.Since(processStarted).Seconds()), "sampledAt": time.Now().UTC().Format(time.RFC3339Nano)}
}
