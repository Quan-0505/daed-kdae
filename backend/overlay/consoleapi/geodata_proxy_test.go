package consoleapi

import (
	"github.com/daeuniverse/dae-wing/dae"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeoSourceProxyPersistenceAndLegacyFormat(t *testing.T) {
	directory := t.TempDir()
	file := filepath.Join(directory, "geodata-sources.json")
	os.WriteFile(file, []byte(`{"geosite":"https://example.test/old.dat"}`), 0600)
	if readSources(directory)["geosite"].UseProxy {
		t.Fatal("legacy default is not direct")
	}
	w := httptest.NewRecorder()
	geodataAPI(w, httptest.NewRequest("POST", "/api/geodata/source", strings.NewReader(`{"kind":"geosite","source":"https://example.test/new.dat","useProxy":true}`)), directory)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	option := readSources(directory)["geosite"]
	if !option.UseProxy || option.Source != "https://example.test/new.dat" {
		t.Fatal("proxy setting lost")
	}
	w = httptest.NewRecorder()
	geodataAPI(w, httptest.NewRequest("POST", "/api/geodata/source", strings.NewReader(`{"kind":"geoip","source":"https://example.test/ip.dat","useProxy":false}`)), directory)
	if readSources(directory)["geoip"].UseProxy || !readSources(directory)["geosite"].UseProxy {
		t.Fatal("kinds are not independent")
	}
}
func TestGeoProxyTransportSelection(t *testing.T) {
	if geoTransport(true) != dae.HttpTransport {
		t.Fatal("proxy did not select engine transport")
	}
	direct, ok := geoTransport(false).(*http.Transport)
	if !ok || direct.Proxy != nil || direct == dae.HttpTransport {
		t.Fatal("direct unexpectedly uses proxy")
	}
	direct.CloseIdleConnections()
}
