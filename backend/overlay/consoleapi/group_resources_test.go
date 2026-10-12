package consoleapi

import (
	"context"
	"encoding/json"
	"github.com/daeuniverse/dae-wing/common"
	"github.com/daeuniverse/dae-wing/db"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGroupResourceEditingAtomicValidationAndConflict(t *testing.T) {
	if err := db.InitDatabase(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	d := db.DB(context.Background())
	sql, _ := d.DB()
	defer sql.Close()
	group := db.Group{Name: "fixture", Policy: "min"}
	d.Create(&group)
	sub := db.Subscription{Link: "https://example.test/subscription"}
	d.Create(&sub)
	node := db.Node{Name: "USA fixture", Link: "fixture", Protocol: "ss", SubscriptionID: &sub.ID}
	d.Create(&node)
	d.Create(&db.GroupSubscription{GroupID: group.ID, SubscriptionID: sub.ID})
	current, err := readGroupResources(d, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	send := func(path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		data, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		groupResourcesAPI(w, httptest.NewRequest("POST", path, strings.NewReader(string(data))))
		return w
	}
	expr := "^USA"
	binding := resourceBinding{string(common.EncodeCursor(sub.ID)), &expr}
	preview := send("/api/groups/preview", groupResources{Subscriptions: []resourceBinding{binding}})
	if preview.Code != 200 || !strings.Contains(preview.Body.String(), "USA fixture") {
		t.Fatal("preview inconsistent", preview.Body.String())
	}
	bad := "(?=USA)"
	request := groupResources{ID: current.ID, Revision: current.Revision, Mode: "subscriptions", Subscriptions: []resourceBinding{{binding.ID, &bad}}}
	if w := send("/api/groups/resources", request); w.Code != 400 {
		t.Fatal("unsupported Go regex accepted")
	}
	after, _ := readGroupResources(d, group.ID)
	if after.Revision != current.Revision || len(after.Subscriptions) != 1 {
		t.Fatal("bad regex modified bindings")
	}
	request.Subscriptions = []resourceBinding{binding}
	if w := send("/api/groups/resources", request); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := send("/api/groups/resources", request); w.Code != 409 {
		t.Fatal("stale editor overwrote group")
	}
	current, _ = readGroupResources(d, group.ID)
	request = groupResources{ID: current.ID, Revision: current.Revision, Mode: "nodes", NodeIDs: []string{string(common.EncodeCursor(node.ID))}}
	if w := send("/api/groups/resources", request); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	current, _ = readGroupResources(d, group.ID)
	if len(current.NodeIDs) != 1 || len(current.Subscriptions) != 1 || *current.Subscriptions[0].Regex != expr {
		t.Fatal("node editing lost subscription filters")
	}
	request.Revision = current.Revision
	request.NodeIDs = []string{string(common.EncodeCursor(9999))}
	if w := send("/api/groups/resources", request); w.Code != 400 {
		t.Fatal("missing node accepted")
	}
	after, _ = readGroupResources(d, group.ID)
	if after.Revision != current.Revision {
		t.Fatal("failed node edit changed bindings")
	}
	request.NodeIDs = []string{}
	if w := send("/api/groups/resources", request); w.Code != 200 {
		t.Fatal("empty selection not saved", w.Body.String())
	}
	after, _ = readGroupResources(d, group.ID)
	if len(after.NodeIDs) != 0 || len(after.Subscriptions) != 1 {
		t.Fatal("empty node selection damaged subscription bindings")
	}
}
