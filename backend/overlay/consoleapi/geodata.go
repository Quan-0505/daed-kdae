package consoleapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/adrg/xdg"
	"github.com/daeuniverse/dae-wing/dae"
	"github.com/daeuniverse/dae/pkg/geodata"
	"google.golang.org/protobuf/proto"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var geoMu sync.Mutex

type GeoInfo struct {
	Kind       string `json:"kind"`
	Version    string `json:"version"`
	Categories int    `json:"categories"`
	Entries    int    `json:"entries"`
	Source     string `json:"source"`
	Bytes      int    `json:"bytes"`
	UseProxy   bool   `json:"useProxy"`
}
type geoCached struct {
	mod  time.Time
	size int64
	info GeoInfo
}

var geoCache = map[string]geoCached{}

func geoPath(directory, kind string) (string, error) {
	bases := []string{}
	if env := os.Getenv("DAE_LOCATION_ASSET"); env != "" {
		bases = append(bases, env, directory, "/usr/local/share/daed", "/usr/share/daed")
	} else {
		bases = append(bases, directory)
		for _, base := range append([]string{xdg.DataHome}, xdg.DataDirs...) {
			bases = append(bases, filepath.Join(base, "daed"))
		}
	}
	for _, base := range bases {
		file := filepath.Join(base, kind+".dat")
		if stat, err := os.Stat(file); err == nil && stat.Mode().IsRegular() {
			return file, nil
		}
	}
	return "", fmt.Errorf("geodata not installed")
}
func geoStats(kind string, data []byte) (int, int, error) {
	if kind == "geoip" {
		var list geodata.GeoIPList
		if err := proto.Unmarshal(data, &list); err != nil || len(list.Entry) == 0 {
			return 0, 0, fmt.Errorf("invalid geoip data")
		}
		total := 0
		for _, e := range list.Entry {
			total += len(e.Cidr)
		}
		return len(list.Entry), total, nil
	}
	var list geodata.GeoSiteList
	if err := proto.Unmarshal(data, &list); err != nil || len(list.Entry) == 0 {
		return 0, 0, fmt.Errorf("invalid geosite data")
	}
	total := 0
	for _, e := range list.Entry {
		total += len(e.Domain)
	}
	return len(list.Entry), total, nil
}

type GeoSource struct {
	Source   string `json:"source"`
	UseProxy bool   `json:"useProxy"`
}

func readSources(directory string) map[string]GeoSource {
	sources := map[string]GeoSource{
		"geosite": {Source: "https://cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/geosite.dat"},
		"geoip":   {Source: "https://cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/geoip.dat"},
	}
	raw := map[string]json.RawMessage{}
	data, _ := os.ReadFile(filepath.Join(directory, "geodata-sources.json"))
	_ = json.Unmarshal(data, &raw)
	for kind, value := range raw {
		var option GeoSource
		if json.Unmarshal(value, &option) != nil {
			_ = json.Unmarshal(value, &option.Source)
		}
		if option.Source != "" {
			sources[kind] = option
		}
	}
	return sources
}
func geoTransport(useProxy bool) http.RoundTripper {
	if useProxy {
		return dae.HttpTransport
	}
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		direct := t.Clone()
		direct.Proxy = nil
		direct.DialContext = geoDirectDial
		return direct
	}
	return http.DefaultTransport
}
func writeJSON(file string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := os.WriteFile(file+".tmp", data, 0600); err != nil {
		return err
	}
	return os.Rename(file+".tmp", file)
}
func geoInfo(directory, kind string) (GeoInfo, error) {
	file, err := geoPath(directory, kind)
	if err != nil {
		return GeoInfo{}, err
	}
	stat, err := os.Stat(file)
	if err != nil || stat.Size() > 64*1024*1024 {
		return GeoInfo{}, fmt.Errorf("geodata unavailable or oversized")
	}
	if cached, ok := geoCache[file]; ok && cached.mod.Equal(stat.ModTime()) && cached.size == stat.Size() {
		option := readSources(directory)[kind]
		cached.info.Source = option.Source
		cached.info.UseProxy = option.UseProxy
		return cached.info, nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return GeoInfo{}, err
	}
	categories, entries, err := geoStats(kind, data)
	if err != nil {
		return GeoInfo{}, err
	}
	versionData, _ := os.ReadFile(file + ".version")
	version := strings.TrimSpace(string(versionData))
	if version == "" {
		version = stat.ModTime().UTC().Format("20060102150405")
	}
	option := readSources(directory)[kind]
	info := GeoInfo{Kind: kind, Version: version, Categories: categories, Entries: entries, Source: option.Source, Bytes: len(data), UseProxy: option.UseProxy}
	geoCache[file] = geoCached{stat.ModTime(), stat.Size(), info}
	return info, nil
}
func geodataAPI(w http.ResponseWriter, r *http.Request, directory string) {
	// The same job can be triggered explicitly for diagnostics without waiting
	// until tomorrow. Normal operation is driven by StartMaintenance.
	if r.URL.Path == "/api/geodata/update-all" {
		if r.Method != "POST" {
			fail(w, 405, "method not allowed")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
		defer cancel()
		if err := runDailyGeo(ctx, directory); err != nil {
			fail(w, 502, err.Error())
			return
		}
		reply(w, 200, map[string]bool{"updated": true})
		return
	}
	geoMu.Lock()
	defer geoMu.Unlock()
	if r.URL.Path == "/api/geodata" && r.Method == "GET" {
		items := []GeoInfo{}
		for _, kind := range []string{"geosite", "geoip"} {
			info, err := geoInfo(directory, kind)
			if err != nil {
				fail(w, 500, "unable to read "+kind)
				return
			}
			items = append(items, info)
		}
		reply(w, 200, map[string]any{"items": items, "schedule": geoSchedule(directory)})
		return
	}
	if r.Method != "POST" && r.Method != "PATCH" {
		fail(w, 405, "method not allowed")
		return
	}
	var request struct {
		Kind     string `json:"kind"`
		Source   string `json:"source"`
		UseProxy *bool  `json:"useProxy"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil || (request.Kind != "geoip" && request.Kind != "geosite") {
		fail(w, 400, "invalid geodata request")
		return
	}
	sources := readSources(directory)
	if r.URL.Path == "/api/geodata/source" {
		u, err := url.Parse(request.Source)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
			fail(w, 400, "source must be an HTTPS URL without embedded credentials")
			return
		}
		option := sources[request.Kind]
		option.Source = request.Source
		if request.UseProxy != nil {
			option.UseProxy = *request.UseProxy
		}
		sources[request.Kind] = option
		if writeJSON(filepath.Join(directory, "geodata-sources.json"), sources) != nil {
			fail(w, 500, "unable to save source")
			return
		}
		reply(w, 200, map[string]bool{"saved": true})
		return
	}
	if r.URL.Path != "/api/geodata/update" || r.Method != "POST" {
		fail(w, 405, "method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	if err := updateManagedGeo(ctx, directory, []string{request.Kind}); err != nil {
		status := 500
		if e, ok := err.(*geoError); ok {
			status = e.status
		}
		fail(w, status, err.Error())
		return
	}
	info, err := geoInfo(directory, request.Kind)
	if err != nil {
		fail(w, 500, "unable to inspect applied geodata")
		return
	}
	reply(w, 200, info)
}
