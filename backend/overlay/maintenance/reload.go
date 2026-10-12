// Package maintenance coordinates background configuration reloads.
package maintenance

import (
	"context"
	"github.com/daeuniverse/dae-wing/db"
	configservice "github.com/daeuniverse/dae-wing/graphql/service/config"
	"github.com/sirupsen/logrus"
	"time"
)

var requests = make(chan struct{}, 1)

// SubscriptionUpdated is called only after a successful database commit.
func SubscriptionUpdated() {
	select {
	case requests <- struct{}{}:
	default:
	}
}

// Reload never starts an engine that the user has stopped.
func Reload(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d := db.DB(ctx)
	var system db.System
	if err := d.FirstOrCreate(&system).Error; err != nil {
		return err
	}
	if !system.Running {
		return nil
	}
	_, err := configservice.Run(d, false)
	return err
}

// Run coalesces subscriptions finishing together and retries a busy/rejected
// reload without changing their cron expressions or discarding saved nodes.
func Run(ctx context.Context) { run(ctx, requests, 10*time.Second, Reload) }
func run(ctx context.Context, queue <-chan struct{}, delay time.Duration, reload func(context.Context) error) {
	var timer *time.Timer
	var tick <-chan time.Time
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	arm := func(d time.Duration) {
		if timer != nil {
			timer.Stop()
		}
		timer = time.NewTimer(d)
		tick = timer.C
	}
	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-queue:
			if tick == nil {
				attempt = 0
				arm(delay)
			}
		case <-tick:
			tick = nil
			if err := ctx.Err(); err != nil {
				return
			}
			if err := reload(ctx); err != nil {
				attempt++
				logrus.WithField("attempt", attempt).WithError(err).Error("Subscription automatic reload failed; previous engine retained")
				if attempt < 3 {
					arm(30 * time.Second)
				}
			} else {
				logrus.Warn("Subscription updates applied; automatic reload completed (stopped engine stays stopped)")
			}
		}
	}
}
