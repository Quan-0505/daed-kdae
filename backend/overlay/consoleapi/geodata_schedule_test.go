package consoleapi

import (
	"bytes"
	"context"
	"fmt"
	"github.com/daeuniverse/dae/pkg/geodata"
	"google.golang.org/protobuf/proto"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestDailyTimeUsesShanghaiAcrossRestartAndUTC(t *testing.T) {
	now := time.Date(2026, 10, 12, 19, 59, 0, 0, time.UTC)
	if next := nextGeoRun(now); !next.Equal(now.Add(time.Minute)) {
		t.Fatal("wrong daily UTC boundary", next)
	}
	if next := nextGeoRun(now.Add(time.Minute)); !next.Equal(now.Add(24*time.Hour + time.Minute)) {
		t.Fatal("job repeats at boundary", next)
	}
	d := t.TempDir()
	status := geoSchedule(d)
	now = time.Now()
	status.LastSuccess = &now
	if err := writeJSON(filepath.Join(d, "geodata-maintenance.json"), status); err != nil {
		t.Fatal(err)
	}
	got := geoSchedule(d)
	if !got.Enabled || got.Time != "04:00" || got.LastSuccess == nil || !got.LastSuccess.Equal(now) {
		t.Fatal("restart lost status")
	}
	for _, kind := range []string{"geosite", "geoip"} {
		if readSources(d)[kind].Source == "" {
			t.Fatal("missing default source")
		}
	}
}

func TestGeodataBatchValidatesBothThenReloadsOnceAndRestoresBoth(t *testing.T) {
	t.Setenv("DAE_LOCATION_ASSET", "")
	dir := t.TempDir()
	ip, _ := proto.Marshal(&geodata.GeoIPList{Entry: []*geodata.GeoIP{{CountryCode: "NEW", Cidr: []*geodata.CIDR{{Ip: []byte{1, 2, 3, 0}, Prefix: 24}}}}})
	site, _ := proto.Marshal(&geodata.GeoSiteList{Entry: []*geodata.GeoSite{{CountryCode: "NEW", Domain: []*geodata.Domain{{Value: "example.com"}}}}})
	var bad atomic.Bool
	bad.Store(true)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/site" {
			w.Write(site)
		} else if bad.Load() {
			w.Write([]byte("invalid"))
		} else {
			w.Write(ip)
		}
	}))
	defer server.Close()
	oldTransport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = oldTransport }()
	writeJSON(filepath.Join(dir, "geodata-sources.json"), map[string]string{"geosite": server.URL + "/site", "geoip": server.URL + "/ip"})
	old := []byte("old file bytes")
	for _, kind := range []string{"geosite", "geoip"} {
		os.WriteFile(filepath.Join(dir, kind+".dat"), old, 0600)
	}
	checkOld := func() {
		t.Helper()
		for _, kind := range []string{"geosite", "geoip"} {
			got, _ := os.ReadFile(filepath.Join(dir, kind+".dat"))
			if !bytes.Equal(got, old) {
				t.Fatal("partial batch installed", kind)
			}
		}
	}
	calls := 0
	reload := func(context.Context) error { calls++; return fmt.Errorf("reject fixture") }
	if err := updateGeodata(context.Background(), dir, []string{"geosite", "geoip"}, reload); err == nil {
		t.Fatal("invalid download accepted")
	}
	checkOld()
	if calls != 0 {
		t.Fatal("reloaded before both downloads validated")
	}
	bad.Store(false)
	if err := updateGeodata(context.Background(), dir, []string{"geosite", "geoip"}, reload); err == nil {
		t.Fatal("rejected engine reported success")
	}
	checkOld()
	if calls != 1 {
		t.Fatal("batch reloaded multiple times")
	}
	if err := updateGeodata(context.Background(), dir, []string{"geosite", "geoip"}, func(context.Context) error { calls++; return nil }); err != nil {
		t.Fatal(err)
	}
	for kind, expected := range map[string][]byte{"geosite": site, "geoip": ip} {
		got, _ := os.ReadFile(filepath.Join(dir, kind+".dat"))
		if !bytes.Equal(got, expected) {
			t.Fatal("batch not committed", kind)
		}
	}
	if calls != 2 {
		t.Fatal("batch did not reload once")
	}
}
