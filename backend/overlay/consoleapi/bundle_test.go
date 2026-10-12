package consoleapi

import (
	"encoding/json"
	"github.com/daeuniverse/dae-wing/dae"
	"github.com/daeuniverse/dae-wing/db"
	"github.com/daeuniverse/dae-wing/pkg/sqlite"
	daeConfig "github.com/daeuniverse/dae/config"
	"github.com/daeuniverse/dae/pkg/geodata"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"strings"
	"testing"
)

func fixture(t *testing.T) *gorm.DB {
	t.Helper()
	d, e := gorm.Open(sqlite.Open(t.TempDir()+"/test.db"), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	e = d.AutoMigrate(&db.Config{}, &db.Dns{}, &db.Routing{}, &db.Node{}, &db.Subscription{}, &db.Group{}, &db.GroupSubscription{}, &db.GroupPolicyParam{}, &db.System{})
	if e != nil {
		t.Fatal(e)
	}
	sql, _ := d.DB()
	t.Cleanup(func() { sql.Close() })
	return d
}
func emptyBundle() Bundle {
	b := Bundle{"daed-kdae", 1, map[string][]map[string]any{}}
	for _, s := range specs {
		b.Tables[s.name] = []map[string]any{}
	}
	return b
}
func TestImportRemapsBindingsPreservesSelectionAndRewritesRouting(t *testing.T) {
	d := fixture(t)
	d.Create(&db.Config{Name: "existing", Global: "global {}", Selected: true})
	b := emptyBundle()
	b.Tables["configs"] = []map[string]any{{"id": int64(1), "name": "global", "global": "global {}", "selected": true, "version": 8}}
	b.Tables["groups"] = []map[string]any{{"id": int64(4), "name": "proxy", "policy": "fixed", "version": 3}}
	b.Tables["nodes"] = []map[string]any{{"id": int64(9), "link": "socks5://127.0.0.1:1080", "name": "fixture", "address": "127.0.0.1:1080", "protocol": "socks5"}}
	b.Tables["group_nodes"] = []map[string]any{{"group_id": int64(4), "node_id": int64(9)}}
	b.Tables["routings"] = []map[string]any{{"id": int64(2), "name": "route", "routing": "routing { fallback: proxy }", "selected": true, "version": 2}}
	data, _ := json.Marshal(b)
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if e := decoder.Decode(&b); e != nil {
		t.Fatal(e)
	}
	counts, e := importBundle(d, b)
	if e != nil {
		t.Fatal(e)
	}
	if counts["groups"] != 1 {
		t.Fatal(counts)
	}
	var groups []db.Group
	d.Preload("Node").Find(&groups)
	if len(groups) != 1 || len(groups[0].Node) != 1 {
		t.Fatal("binding lost")
	}
	var route db.Routing
	d.First(&route)
	if !strings.Contains(route.Routing, groups[0].Name) {
		t.Fatal("outbound not renamed")
	}
	var selected int64
	d.Model(&db.Config{}).Where("selected = ?", true).Count(&selected)
	if selected != 1 {
		t.Fatal("selected configuration changed")
	}
	var imported db.Config
	d.Where("name != ?", "existing").First(&imported)
	if imported.Selected || imported.Version != 1 {
		t.Fatal("imported selection/version incorrect")
	}
	out, e := exportBundle(d)
	if e != nil {
		t.Fatal(e)
	}
	if _, ok := out.Tables["users"]; ok {
		t.Fatal("account exported")
	}
}
func TestInvalidBindingRollsBackWholeImport(t *testing.T) {
	d := fixture(t)
	b := emptyBundle()
	b.Tables["configs"] = []map[string]any{{"id": 1, "name": "partial", "global": "global {}", "selected": false, "version": 1}}
	b.Tables["group_nodes"] = []map[string]any{{"group_id": 999, "node_id": 888}}
	if _, e := importBundle(d, b); e == nil {
		t.Fatal("bad binding accepted")
	}
	var count int64
	d.Model(&db.Config{}).Count(&count)
	if count != 0 {
		t.Fatal("partial import committed")
	}
}
func TestInvalidConfigurationRejected(t *testing.T) {
	b := emptyBundle()
	b.Tables["dns"] = []map[string]any{{"id": 1, "dns": "dns { garbage ! }"}}
	if validateBundle(b) == nil {
		t.Fatal("invalid DNS accepted")
	}
}
func TestGeoValidationAndCounts(t *testing.T) {
	if _, _, e := geoStats("geoip", []byte("invalid")); e == nil {
		t.Fatal("invalid file accepted")
	}
	data, e := proto.Marshal(&geodata.GeoIPList{Entry: []*geodata.GeoIP{{CountryCode: "TEST", Cidr: []*geodata.CIDR{{Ip: []byte{1, 2, 3, 0}, Prefix: 24}}}}})
	if e != nil {
		t.Fatal(e)
	}
	categories, entries, e := geoStats("geoip", data)
	if e != nil || categories != 1 || entries != 1 {
		t.Fatal(categories, entries, e)
	}
}

func TestDaeSnapshotImportRoundTrip(t *testing.T) {
	source := `global {}
dns {}
node { test: 'socks5://127.0.0.1:1080' }
group { proxy { filter: name(test) policy: fixed } }
routing { fallback: proxy }`
	b, e := daeBundle(source)
	if e != nil {
		t.Fatal(e)
	}
	d := fixture(t)
	if _, e = importBundle(d, b); e != nil {
		t.Fatal(e)
	}
	var groups []db.Group
	d.Preload("Node").Find(&groups)
	if len(groups) != 1 || len(groups[0].Node) != 1 {
		t.Fatal("dae group binding lost")
	}
	if _, e = daeBundle("include { /etc/passwd }\n" + source); e == nil {
		t.Fatal("include accepted")
	}
}

func TestExportNumericAndWhitespaceNodeNamesRoundTrip(t *testing.T) {
	c, e := dae.ParseConfig(nil, nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	c.Node = []daeConfig.KeyableString{"1.fixture node:socks5://127.0.0.1:1080"}
	data, e := renderSafeDAE(c)
	if e != nil {
		t.Fatal(e)
	}
	b, e := daeBundle(string(data))
	if e != nil {
		t.Fatal(e)
	}
	if len(b.Tables["nodes"]) != 1 {
		t.Fatal("exported node lost")
	}
}
