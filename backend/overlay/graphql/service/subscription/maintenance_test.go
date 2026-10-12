package subscription

import (
	"context"
	"github.com/daeuniverse/dae-wing/dae"
	"github.com/daeuniverse/dae-wing/db"
	"github.com/daeuniverse/dae-wing/maintenance"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestScheduledSubscriptionReloadsAfterCommitAndFailurePreservesNodes(t *testing.T) {
	if err := db.InitDatabase(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	d := db.DB(context.Background())
	sql, _ := d.DB()
	defer sql.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); maintenance.Run(ctx) }()
	defer func() { cancel(); <-done }()
	d.Create(&db.Config{Name: "test", Global: "global {}", Selected: true})
	d.Create(&db.Dns{Name: "test", Dns: "dns {}", Selected: true})
	d.Create(&db.Routing{Name: "test", Routing: "routing { fallback: direct }", Selected: true})
	d.Create(&db.System{Running: true})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte("ss://YWVzLTEyOC1nY206cGFzcw==@127.0.0.1:19000#fixture"))
	}))
	defer server.Close()
	sub := db.Subscription{Link: server.URL + "/ok", CronEnable: true, CronExp: "0 0 1 1 *"}
	if err := d.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	AddUpdateScheduler(context.Background(), sub.ID)
	defer RemoveUpdateScheduler(sub.ID)
	schedulerMu.RLock()
	scheduler := schedulerCache[sub.ID]
	schedulerMu.RUnlock()
	if scheduler == nil {
		t.Fatal("cron job was not installed")
	}
	scheduler.RunAll() // Execute the actual registered cron callback now.
	select {
	case msg := <-dae.ChReloadConfigs:
		var count int64
		d.Model(&db.Node{}).Where("subscription_id = ?", sub.ID).Count(&count)
		if count != 1 {
			msg.Callback <- nil
			t.Fatal("reload ran before node commit", count)
		}
		msg.Callback <- nil
	case <-time.After(15 * time.Second):
		t.Fatal("successful scheduled update did not reload")
	}
	d.Model(&sub).Update("link", server.URL+"/fail")
	if _, err := ScheduledUpdate(context.Background(), sub.ID); err == nil {
		t.Fatal("failed subscription reported success")
	}
	var count int64
	d.Model(&db.Node{}).Where("subscription_id = ?", sub.ID).Count(&count)
	if count != 1 {
		t.Fatal("failed update discarded nodes")
	}
	select {
	case msg := <-dae.ChReloadConfigs:
		msg.Callback <- nil
		t.Fatal("failed update requested reload")
	case <-time.After(11 * time.Second):
	}
}
