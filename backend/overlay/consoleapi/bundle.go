package consoleapi

import (
	"encoding/json"
	"fmt"
	"github.com/daeuniverse/dae-wing/dae"
	"github.com/daeuniverse/dae-wing/db"
	configservice "github.com/daeuniverse/dae-wing/graphql/service/config"
	daeConfig "github.com/daeuniverse/dae/config"
	"gorm.io/gorm"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type tableSpec struct {
	name    string
	columns string
}

var specs = []tableSpec{{"configs", "id,name,global,selected,version"}, {"dns", "id,name,dns,selected,version"}, {"routings", "id,name,routing,selected,version"}, {"subscriptions", "id,updated_at,link,cron_exp,cron_enable,status,info,tag"}, {"nodes", "id,link,name,address,protocol,tag,subscription_id"}, {"groups", "id,name,policy,version,system_id"}, {"group_policy_params", "id,key,value,group_id"}, {"group_nodes", "group_id,node_id"}, {"group_subscriptions", "group_id,subscription_id,name_filter_regex"}}

type Bundle struct {
	Format  string                      `json:"format"`
	Version int                         `json:"version"`
	Tables  map[string][]map[string]any `json:"tables"`
}

func exportBundle(d *gorm.DB) (Bundle, error) {
	bundle := Bundle{"daed-kdae", 1, map[string][]map[string]any{}}
	for _, spec := range specs {
		rows := []map[string]any{}
		if err := d.Table(spec.name).Select(spec.columns).Find(&rows).Error; err != nil {
			return bundle, err
		}
		bundle.Tables[spec.name] = rows
	}
	return bundle, nil
}
func number(v any) (int64, error) {
	switch n := v.(type) {
	case json.Number:
		return n.Int64()
	case float64:
		if n == float64(int64(n)) {
			return int64(n), nil
		}
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	}
	return 0, fmt.Errorf("invalid resource ID")
}
func text(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
func validateBundle(bundle Bundle) error {
	if bundle.Format != "daed-kdae" || bundle.Version != 1 {
		return fmt.Errorf("unsupported backup format")
	}
	total := 0
	for _, spec := range specs {
		rows, ok := bundle.Tables[spec.name]
		if !ok {
			return fmt.Errorf("missing table %s", spec.name)
		}
		total += len(rows)
		if total > 20000 {
			return fmt.Errorf("backup too large")
		}
		allowed := map[string]bool{}
		for _, name := range strings.Split(spec.columns, ",") {
			allowed[name] = true
		}
		seen := map[int64]bool{}
		for _, row := range rows {
			for name := range row {
				if !allowed[name] {
					return fmt.Errorf("unexpected field in %s", spec.name)
				}
			}
			if _, ok := row["id"]; ok {
				id, err := number(row["id"])
				if err != nil || id < 1 || seen[id] {
					return fmt.Errorf("invalid or duplicate ID in %s", spec.name)
				}
				seen[id] = true
			}
		}
	}
	for _, row := range bundle.Tables["configs"] {
		g := text(row["global"])
		if _, err := dae.ParseConfig(&g, nil, nil); err != nil {
			return fmt.Errorf("invalid global configuration")
		}
	}
	for _, row := range bundle.Tables["dns"] {
		dns := text(row["dns"])
		if _, err := dae.ParseConfig(nil, &dns, nil); err != nil {
			return fmt.Errorf("invalid DNS configuration")
		}
	}
	for _, row := range bundle.Tables["routings"] {
		route := text(row["routing"])
		if _, err := dae.ParseConfig(nil, nil, &route); err != nil {
			return fmt.Errorf("invalid routing configuration")
		}
	}
	return nil
}
func renamedOutbound(name string, names map[string]string) string {
	if mapped, ok := names[name]; ok {
		return mapped
	}
	if strings.HasPrefix(name, "must_") {
		if mapped, ok := names[strings.TrimPrefix(name, "must_")]; ok {
			return "must_" + mapped
		}
	}
	return name
}
func renameRouting(source string, names map[string]string) (string, error) {
	g := dae.EmptyGlobalSection
	dns := dae.EmptyDnsSection
	c, err := dae.ParseConfig(&g, &dns, &source)
	if err != nil {
		return "", err
	}
	for _, rule := range c.Routing.Rules {
		rule.Outbound.Name = renamedOutbound(rule.Outbound.Name, names)
	}
	if fallback, err := daeConfig.ParseFunctionOrString(c.Routing.Fallback); err == nil {
		fallback.Name = renamedOutbound(fallback.Name, names)
		c.Routing.Fallback = fallback
	}
	marshaler := daeConfig.Marshaller{IndentSpace: 2}
	if err := marshaler.MarshalSection("routing", reflect.ValueOf(c.Routing), 0); err != nil {
		return "", err
	}
	return string(marshaler.Bytes()), nil
}
func importBundle(d *gorm.DB, bundle Bundle) (map[string]int, error) {
	if err := validateBundle(bundle); err != nil {
		return nil, err
	}
	counts := map[string]int{}
	mapping := map[string]map[int64]int64{}
	names := map[string]string{}
	prefix := "import" + strconv.FormatInt(time.Now().UnixMilli(), 36) + "_"
	for _, row := range bundle.Tables["groups"] {
		name := text(row["name"])
		if name == "" {
			return nil, fmt.Errorf("group name missing")
		}
		names[name] = prefix + name
	}
	err := d.Transaction(func(tx *gorm.DB) error {
		for _, spec := range specs {
			mapping[spec.name] = map[int64]int64{}
			for _, original := range bundle.Tables[spec.name] {
				row := map[string]any{}
				for k, v := range original {
					if k != "id" {
						if n, ok := v.(json.Number); ok {
							if number, err := n.Int64(); err == nil {
								v = number
							} else {
								v = n.String()
							}
						}
						row[k] = v
					}
				}
				if _, ok := row["selected"]; ok {
					row["selected"] = false
				}
				if _, ok := row["version"]; ok {
					row["version"] = int64(1)
				}
				switch spec.name {
				case "configs", "dns", "routings", "groups":
					row["name"] = prefix + text(row["name"])
				}
				if tag := text(row["tag"]); tag != "" {
					row["tag"] = prefix + tag
				}
				if spec.name == "groups" {
					row["system_id"] = nil
				}
				if spec.name == "subscriptions" {
					row["cron_enable"] = false
				}
				if spec.name == "routings" {
					route, err := renameRouting(text(row["routing"]), names)
					if err != nil {
						return err
					}
					row["routing"] = route
				}
				for field, table := range map[string]string{"subscription_id": "subscriptions", "node_id": "nodes", "group_id": "groups"} {
					if v, ok := row[field]; ok && v != nil {
						id, err := number(v)
						if err != nil {
							return err
						}
						mapped, exists := mapping[table][id]
						if !exists {
							return fmt.Errorf("missing %s binding", field)
						}
						row[field] = mapped
					}
				}
				keys := []string{}
				values := []any{}
				for _, key := range strings.Split(spec.columns, ",") {
					if key == "id" {
						continue
					}
					if value, ok := row[key]; ok {
						keys = append(keys, "\""+key+"\"")
						values = append(values, value)
					}
				}
				placeholders := strings.TrimSuffix(strings.Repeat("?,", len(keys)), ",")
				if err := tx.Exec("INSERT INTO "+spec.name+" ("+strings.Join(keys, ",")+") VALUES ("+placeholders+")", values...).Error; err != nil {
					return fmt.Errorf("failed to import %s", spec.name)
				}
				if v, ok := original["id"]; ok {
					id, err := number(v)
					if err != nil {
						return err
					}
					var newID int64
					if err := tx.Raw("SELECT last_insert_rowid()").Scan(&newID).Error; err != nil {
						return err
					}
					mapping[spec.name][id] = newID
				}
				counts[spec.name]++
			}
		}
		return nil
	})
	return counts, err
}
func bundleAPI(w http.ResponseWriter, r *http.Request) {
	d := db.DB(r.Context())
	if r.URL.Path == "/api/config/dae" {
		if r.Method == "POST" {
			data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10*1024*1024))
			if err != nil {
				fail(w, 400, "dae file too large")
				return
			}
			bundle, err := daeBundle(string(data))
			if err != nil {
				fail(w, 400, err.Error())
				return
			}
			if r.URL.Query().Get("preview") == "true" {
				counts := map[string]int{}
				for table, rows := range bundle.Tables {
					counts[table] = len(rows)
				}
				reply(w, 200, map[string]any{"counts": counts})
				return
			}
			counts, err := importBundle(d, bundle)
			if err != nil {
				fail(w, 400, err.Error())
				return
			}
			reply(w, 200, map[string]any{"counts": counts, "imported": true})
			return
		}
		if r.Method != "GET" {
			fail(w, 405, "method not allowed")
			return
		}
		plan, err := configservice.BuildSelected(d)
		if err != nil {
			fail(w, 400, "unable to render selected configuration")
			return
		}
		data, err := renderSafeDAE(plan.Config)
		if err != nil {
			fail(w, 500, "unable to render configuration")
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=daed-config.dae")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(data)
		return
	}
	if r.Method == "GET" && r.URL.Path == "/api/config/bundle" {
		var bundle Bundle
		err := d.Transaction(func(tx *gorm.DB) error { var err error; bundle, err = exportBundle(tx); return err })
		if err != nil {
			fail(w, 500, "backup export failed")
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename=daed-kdae-backup.json")
		reply(w, 200, bundle)
		return
	}
	if r.Method != "POST" {
		fail(w, 405, "method not allowed")
		return
	}
	var bundle Bundle
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10*1024*1024))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if decoder.Decode(&bundle) != nil {
		fail(w, 400, "invalid backup JSON")
		return
	}
	if err := validateBundle(bundle); err != nil {
		fail(w, 400, err.Error())
		return
	}
	if r.URL.Path == "/api/config/bundle/preview" {
		counts := map[string]int{}
		for table, rows := range bundle.Tables {
			counts[table] = len(rows)
		}
		reply(w, 200, map[string]any{"counts": counts, "mode": "add", "note": "Existing resources and running configuration are preserved; imported subscription schedules start disabled."})
		return
	}
	counts, err := importBundle(d, bundle)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	reply(w, 200, map[string]any{"counts": counts, "imported": true})
}
