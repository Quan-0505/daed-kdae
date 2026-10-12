package maintenance

import (
	"context"
	"github.com/daeuniverse/dae-wing/db"
	"sync/atomic"
	"testing"
	"time"
)

func TestCoalesceAndCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	queue := make(chan struct{}, 8)
	var calls atomic.Int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		run(ctx, queue, 30*time.Millisecond, func(context.Context) error { calls.Add(1); return nil })
	}()
	for i := 0; i < 5; i++ {
		queue <- struct{}{}
	}
	deadline := time.After(time.Second)
	for calls.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("reload not called")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	time.Sleep(70 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatal("same batch reloaded repeatedly")
	}
	queue <- struct{}{}
	cancel()
	<-done
	if calls.Load() != 1 {
		t.Fatal("reload ran after cancellation")
	}
}

func TestStoppedEngineStaysStopped(t *testing.T) {
	if err := db.InitDatabase(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	d := db.DB(context.Background())
	sql, _ := d.DB()
	defer sql.Close()
	if err := d.Create(&db.System{Running: false}).Error; err != nil {
		t.Fatal(err)
	}
	if err := Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	var system db.System
	d.First(&system)
	if system.Running {
		t.Fatal("stopped engine started")
	}
}
