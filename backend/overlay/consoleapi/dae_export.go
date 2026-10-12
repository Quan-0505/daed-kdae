package consoleapi

import (
	"fmt"
	daeConfig "github.com/daeuniverse/dae/config"
	"github.com/daeuniverse/dae/pkg/config_parser"
	"strconv"
	"strings"
)

// The core marshaler assumes declaration keys are identifiers. Subscription node
// names can start with digits or contain spaces. Quoting an entire list entry is
// the parser-supported representation that preserves its key and value.
func renderSafeDAE(c *daeConfig.Config) ([]byte, error) {
	raw, err := c.Marshal(2)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")
	stack := []string{}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "}" {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		}
		if strings.HasSuffix(trimmed, " {") {
			stack = append(stack, strings.TrimSuffix(trimmed, " {"))
			continue
		}
		if len(stack) == 0 {
			continue
		}
		switch stack[len(stack)-1] {
		case "node", "subscription", "upstream", "fixed_domain_ttl":
		default:
			continue
		}
		if strings.HasPrefix(trimmed, "\"") {
			continue
		}
		colon := strings.Index(trimmed, ":")
		if colon < 1 {
			continue
		}
		value, err := strconv.Unquote(trimmed[colon+1:])
		if err != nil {
			return nil, fmt.Errorf("invalid list entry")
		}
		lines[i] = line[:len(line)-len(strings.TrimLeft(line, " \t"))] + strconv.Quote(trimmed[:colon]+":"+value)
	}
	data := []byte(strings.Join(lines, "\n"))
	sections, err := config_parser.Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("configuration export did not parse")
	}
	if _, err = daeConfig.New(sections); err != nil {
		return nil, fmt.Errorf("configuration export did not validate")
	}
	return data, nil
}
