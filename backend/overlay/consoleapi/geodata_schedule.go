package consoleapi

import (
	"context"
	"encoding/json"
	"github.com/daeuniverse/dae-wing/maintenance"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"time"
	_ "time/tzdata"
)

type GeoSchedule struct {
	Enabled     bool       `json:"enabled"`
	Time        string     `json:"time"`
	Timezone    string     `json:"timezone"`
	NextRun     time.Time  `json:"nextRun"`
	LastAttempt *time.Time `json:"lastAttempt,omitempty"`
	LastSuccess *time.Time `json:"lastSuccess,omitempty"`
	Error       string     `json:"error,omitempty"`
}

func nextGeoRun(now time.Time) time.Time {
	location, _ := time.LoadLocation("Asia/Shanghai")
	local := now.In(location)
	next := time.Date(local.Year(), local.Month(), local.Day(), 4, 0, 0, 0, location)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
func geoSchedule(directory string) GeoSchedule {
	var result GeoSchedule
	data, _ := os.ReadFile(filepath.Join(directory, "geodata-maintenance.json"))
	_ = json.Unmarshal(data, &result)
	result.Enabled, result.Time, result.Timezone = true, "04:00", "Asia/Shanghai"
	result.NextRun = nextGeoRun(time.Now())
	return result
}
func runDailyGeo(ctx context.Context, directory string) error {
	geoMu.Lock()
	defer geoMu.Unlock()
	status := geoSchedule(directory)
	now := time.Now()
	status.LastAttempt = &now
	err := updateManagedGeo(ctx, directory, []string{"geosite", "geoip"})
	if err == nil {
		status.LastSuccess = &now
		status.Error = ""
	} else {
		status.Error = err.Error()
	}
	if saveErr := writeJSON(filepath.Join(directory, "geodata-maintenance.json"), status); saveErr != nil {
		logrus.WithError(saveErr).Error("Unable to persist geodata automatic update status")
	}
	return err
}

// StartMaintenance is called after the engine's saved running state is restored.
// Stop cancels downloads and waits before the engine receives its shutdown message.
func StartMaintenance(directory string) func() {
	ctx, cancel := context.WithCancel(context.Background())
	doneGeo, doneReload := make(chan struct{}), make(chan struct{})
	go func() { defer close(doneReload); maintenance.Run(ctx) }()
	go func() {
		defer close(doneGeo)
		for {
			timer := time.NewTimer(time.Until(nextGeoRun(time.Now())))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				jobCtx, stop := context.WithTimeout(ctx, 3*time.Minute)
				err := runDailyGeo(jobCtx, directory)
				stop()
				if err != nil {
					logrus.WithError(err).Error("Daily geodata update failed; previous data retained")
				}
			}
		}
	}()
	return func() { cancel(); <-doneGeo; <-doneReload }
}
