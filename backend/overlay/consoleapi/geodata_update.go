package consoleapi

import (
	"context"
	"fmt"
	"github.com/daeuniverse/dae-wing/maintenance"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type geoError struct {
	status  int
	message string
}

func (e *geoError) Error() string               { return e.message }
func geoFailure(code int, message string) error { return &geoError{code, message} }

func downloadGeo(ctx context.Context, kind string, option GeoSource) ([]byte, error) {
	if option.Source == "" {
		return nil, geoFailure(400, "configure a geodata source first")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", option.Source, nil)
	if err != nil {
		return nil, geoFailure(400, "invalid source")
	}
	transport := geoTransport(option.UseProxy)
	if !option.UseProxy {
		if t, ok := transport.(*http.Transport); ok {
			defer t.CloseIdleConnections()
		}
	}
	client := http.Client{Transport: transport, Timeout: 90 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Scheme != "https" || len(via) > 5 {
			return fmt.Errorf("unsafe redirect")
		}
		return nil
	}}
	response, err := client.Do(req)
	if err != nil {
		return nil, geoFailure(502, kind+" download failed")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, geoFailure(502, kind+" source returned non-success status")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 64*1024*1024+1))
	if err != nil || len(data) > 64*1024*1024 {
		return nil, geoFailure(400, "download oversized or interrupted")
	}
	if _, _, err := geoStats(kind, data); err != nil {
		return nil, geoFailure(400, "downloaded file is not valid "+kind)
	}
	return data, nil
}

// Caller holds geoMu. Download and validate the complete batch before changing
// any installed files. A rejected engine candidate restores the complete batch.
func updateGeodata(ctx context.Context, directory string, kinds []string, reload func(context.Context) error) error {
	if env := os.Getenv("DAE_LOCATION_ASSET"); env != "" && filepath.Clean(env) != filepath.Clean(directory) {
		return geoFailure(400, "DAE_LOCATION_ASSET overrides the managed directory; update that directory externally")
	}
	type change struct {
		file             string
		next, previous   []byte
		existed, applied bool
	}
	changes := make([]change, 0, len(kinds))
	sources := readSources(directory)
	for _, kind := range kinds {
		data, err := downloadGeo(ctx, kind, sources[kind])
		if err != nil {
			return err
		}
		file := filepath.Join(directory, kind+".dat")
		previous, err := os.ReadFile(file)
		if err != nil && !os.IsNotExist(err) {
			return geoFailure(500, "unable to back up existing geodata")
		}
		changes = append(changes, change{file: file, next: data, previous: previous, existed: err == nil})
	}
	rollback := func(cause error) error {
		failed := false
		for _, c := range changes {
			_ = os.Remove(c.file + ".staged")
			if !c.applied {
				continue
			}
			delete(geoCache, c.file)
			var err error
			if c.existed {
				err = os.WriteFile(c.file+".restore", c.previous, 0600)
				if err == nil {
					err = os.Rename(c.file+".restore", c.file)
				}
			} else {
				err = os.Remove(c.file)
			}
			if err != nil {
				failed = true
			}
		}
		if failed {
			return geoFailure(500, "geodata file restoration failed; restore the file before restarting")
		}
		return cause
	}
	for i := range changes {
		c := &changes[i]
		if err := ctx.Err(); err != nil {
			return rollback(err)
		}
		if err := os.WriteFile(c.file+".staged", c.next, 0600); err != nil {
			return rollback(geoFailure(500, "unable to stage update"))
		}
		if err := os.Rename(c.file+".staged", c.file); err != nil {
			return rollback(geoFailure(500, "unable to commit update"))
		}
		c.applied = true
	}
	if err := reload(ctx); err != nil {
		return rollback(geoFailure(400, "engine rejected geodata; previous files restored"))
	}
	for _, c := range changes {
		version := time.Now().UTC().Format("20060102150405")
		if err := os.WriteFile(c.file+".version", []byte(version), 0600); err != nil {
			_ = os.Remove(c.file + ".version")
			logrus.Warn("Geodata updated; version metadata could not be saved")
		}
		delete(geoCache, c.file)
	}
	logrus.WithField("kinds", kinds).Warn("Geodata updates applied; automatic reload completed (stopped engine stays stopped)")
	return nil
}

func updateManagedGeo(ctx context.Context, directory string, kinds []string) error {
	return updateGeodata(ctx, directory, kinds, maintenance.Reload)
}
