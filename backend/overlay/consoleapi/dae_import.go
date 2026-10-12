package consoleapi

import (
	"fmt"
	"github.com/daeuniverse/dae-wing/db"
	daeConfig "github.com/daeuniverse/dae/config"
	"github.com/daeuniverse/dae/pkg/config_parser"
	"reflect"
	"strings"
)

// DAE imports are snapshots. Unsupported dynamic filters are rejected before any write.
func daeBundle(source string) (Bundle, error) {
	sections, err := config_parser.Parse(source)
	if err != nil {
		return Bundle{}, fmt.Errorf("invalid dae syntax")
	}
	for _, section := range sections {
		switch section.Name {
		case "global", "dns", "routing", "node", "group", "subscription":
		default:
			return Bundle{}, fmt.Errorf("unsupported dae section; use a self-contained file without includes")
		}
	}
	c, err := daeConfig.New(sections)
	if err != nil {
		return Bundle{}, fmt.Errorf("invalid dae configuration")
	}
	if len(c.Subscription) > 0 {
		return Bundle{}, fmt.Errorf("dae subscription imports are not supported; use a JSON backup or import subscription links separately")
	}
	b := Bundle{"daed-kdae", 1, map[string][]map[string]any{}}
	for _, s := range specs {
		b.Tables[s.name] = []map[string]any{}
	}
	for _, section := range []struct {
		table, name, field string
		value              any
	}{{"configs", "global", "global", c.Global}, {"dns", "dns", "dns", c.Dns}, {"routings", "routing", "routing", c.Routing}} {
		m := daeConfig.Marshaller{IndentSpace: 2}
		if err := m.MarshalSection(section.name, reflect.ValueOf(section.value), 0); err != nil {
			return b, fmt.Errorf("unable to render dae section")
		}
		b.Tables[section.table] = []map[string]any{{"id": 1, "name": section.name, "selected": false, "version": 1, section.field: string(m.Bytes())}}
	}
	tags := map[string]int{}
	for i, value := range c.Node {
		link := string(value)
		tag := ""
		if colon := strings.Index(link, ":"); colon >= 0 && !strings.HasPrefix(link[colon:], "://") {
			tag = link[:colon]
			link = link[colon+1:]
		}
		node, err := db.NewNodeModel(link, nil, nil)
		if err != nil {
			return b, fmt.Errorf("invalid node in dae file")
		}
		name := tag
		if name == "" {
			name = node.Name
		}
		if _, exists := tags[name]; exists {
			return b, fmt.Errorf("duplicate dae node name")
		}
		tags[name] = i + 1
		b.Tables["nodes"] = append(b.Tables["nodes"], map[string]any{"id": i + 1, "link": link, "name": node.Name, "address": node.Address, "protocol": node.Protocol, "tag": nil})
	}
	paramID := 0
	for i, group := range c.Group {
		if len(group.TcpCheckUrl) > 0 || group.TcpCheckHttpMethod != "" || len(group.UdpCheckDns) > 0 || group.CheckInterval != 0 || group.CheckTolerance != 0 {
			return b, fmt.Errorf("group-specific health checks cannot be imported; use global checks")
		}
		policies, err := daeConfig.ParseFunctionListOrString(group.Policy)
		if err != nil || len(policies) != 1 || policies[0].Not {
			return b, fmt.Errorf("unsupported group policy")
		}
		policy := policies[0]
		b.Tables["groups"] = append(b.Tables["groups"], map[string]any{"id": i + 1, "name": group.Name, "policy": policy.Name, "version": 1})
		for _, p := range policy.Params {
			paramID++
			b.Tables["group_policy_params"] = append(b.Tables["group_policy_params"], map[string]any{"id": paramID, "key": p.Key, "value": p.Val, "group_id": i + 1})
		}
		bound := map[int]bool{}
		if len(group.Filter) == 0 {
			for _, id := range tags {
				bound[id] = true
			}
		}
		for _, filter := range group.Filter {
			if len(filter) != 1 || filter[0].Name != "name" || filter[0].Not {
				return b, fmt.Errorf("only explicit name group filters can be imported; export a Go dae snapshot or use JSON backup")
			}
			for _, p := range filter[0].Params {
				if p.Key != "" {
					return b, fmt.Errorf("regex group filters require JSON backup")
				}
				id, exists := tags[p.Val]
				if !exists {
					return b, fmt.Errorf("group filter refers to missing node")
				}
				bound[id] = true
			}
		}
		for id := range bound {
			b.Tables["group_nodes"] = append(b.Tables["group_nodes"], map[string]any{"group_id": i + 1, "node_id": id})
		}
	}
	return b, validateBundle(b)
}
