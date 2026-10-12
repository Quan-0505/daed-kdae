package consoleapi

import (
	"bytes"
	"context"
	"fmt"
	"github.com/daeuniverse/dae-wing/dae"
	"github.com/daeuniverse/dae-wing/db"
	"github.com/daeuniverse/dae/pkg/geodata"
	"google.golang.org/protobuf/proto"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeodataRejectRestoresPreviousAndAcceptCommits(t *testing.T) {
	t.Setenv("DAE_LOCATION_ASSET", "")
	directory := t.TempDir()
	if err := db.InitDatabase(directory); err != nil {
		t.Fatal(err)
	}
	d := db.DB(context.Background())
	sql, _ := d.DB()
	t.Cleanup(func() { sql.Close() })
	d.Create(&db.Config{Name: "test", Global: "global {}", Selected: true})
	d.Create(&db.Dns{Name: "test", Dns: "dns {}", Selected: true})
	d.Create(&db.Routing{Name: "test", Routing: "routing { fallback: direct }", Selected: true})
	d.Create(&db.System{Running: true})
	old, _ := proto.Marshal(&geodata.GeoIPList{Entry: []*geodata.GeoIP{{CountryCode: "OLD", Cidr: []*geodata.CIDR{{Ip: []byte{1, 2, 3, 0}, Prefix: 24}}}}})
	next, _ := proto.Marshal(&geodata.GeoIPList{Entry: []*geodata.GeoIP{{CountryCode: "NEW", Cidr: []*geodata.CIDR{{Ip: []byte{4, 5, 6, 0}, Prefix: 24}}}}})
	file := filepath.Join(directory, "geoip.dat")
	if err := os.WriteFile(file, old, 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(next) }))
	defer server.Close()
	transport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = transport }()
	if err := writeJSON(filepath.Join(directory, "geodata-sources.json"), map[string]string{"geoip": server.URL}); err != nil {
		t.Fatal(err)
	}
	response := make(chan bool, 1)
	go func() {
		msg := <-dae.ChReloadConfigs
		msg.Callback <- fmt.Errorf("fixture rejects candidate")
		response <- true
	}()
	w := httptest.NewRecorder()
	geodataAPI(w, httptest.NewRequest("POST", "/api/geodata/update", strings.NewReader(`{"kind":"geoip"}`)), directory)
	if w.Code != 400 {
		t.Fatal(w.Code, w.Body.String())
	}
	<-response
	got, _ := os.ReadFile(file)
	if !bytes.Equal(got, old) {
		t.Fatal("previous geodata not restored")
	}
	if err := os.Mkdir(file+".restore", 0700); err != nil {
		t.Fatal(err)
	}
	go func() {
		msg := <-dae.ChReloadConfigs
		msg.Callback <- fmt.Errorf("fixture rejects candidate")
		response <- true
	}()
	w = httptest.NewRecorder()
	geodataAPI(w, httptest.NewRequest("POST", "/api/geodata/update", strings.NewReader(`{"kind":"geoip"}`)), directory)
	if w.Code != 500 || !strings.Contains(w.Body.String(), "restoration failed") {
		t.Fatal("failed restoration reported as success", w.Code)
	}
	<-response
	if err := os.Remove(file + ".restore"); err != nil {
		t.Fatal(err)
	}
	d.Model(&db.System{}).Where("1=1").Update("running", false)
	w = httptest.NewRecorder()
	geodataAPI(w, httptest.NewRequest("POST", "/api/geodata/update", strings.NewReader(`{"kind":"geoip"}`)), directory)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	got, _ = os.ReadFile(file)
	if !bytes.Equal(got, next) {
		t.Fatal("validated update not committed")
	}
	w = httptest.NewRecorder()
	geodataAPI(w, httptest.NewRequest("POST", "/api/geodata/source", strings.NewReader(`{"kind":"geoip","source":"http://example.test/file"}`)), directory)
	if w.Code != 400 {
		t.Fatal("insecure source accepted")
	}
}
