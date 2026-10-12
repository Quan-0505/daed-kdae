package consoleapi

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/daeuniverse/dae-wing/common"
	"github.com/daeuniverse/dae-wing/db"
	"github.com/graph-gophers/graphql-go"
	"gorm.io/gorm"
	"net/http"
	"regexp"
	"strings"
)

type resourceBinding struct {
	ID    string  `json:"id"`
	Regex *string `json:"regex"`
}
type groupResources struct {
	ID            string            `json:"id"`
	Revision      string            `json:"revision"`
	Mode          string            `json:"mode"`
	NodeIDs       []string          `json:"nodeIDs"`
	Subscriptions []resourceBinding `json:"subscriptions"`
}

func cursorID(value string) (uint, error) { return common.DecodeCursor(graphql.ID(value)) }
func readGroupResources(d *gorm.DB, id uint) (groupResources, error) {
	var group db.Group
	if err := d.Preload("Node").Preload("SubscriptionBindings").First(&group, id).Error; err != nil {
		return groupResources{}, err
	}
	result := groupResources{ID: string(common.EncodeCursor(id)), NodeIDs: []string{}, Subscriptions: []resourceBinding{}}
	for _, node := range group.Node {
		result.NodeIDs = append(result.NodeIDs, string(common.EncodeCursor(node.ID)))
	}
	for _, binding := range group.SubscriptionBindings {
		result.Subscriptions = append(result.Subscriptions, resourceBinding{string(common.EncodeCursor(binding.SubscriptionID)), binding.NameFilterRegex})
	}
	// Include group version so concurrent changes through the existing GraphQL
	// editor are also detected before replacing any bindings.
	data, _ := json.Marshal(struct {
		Version       uint
		Nodes         []string
		Subscriptions []resourceBinding
	}{group.Version, result.NodeIDs, result.Subscriptions})
	result.Revision = fmt.Sprintf("%x", sha256.Sum256(data))
	return result, nil
}

type previewNode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
}
type previewSubscription struct {
	ID    string        `json:"id"`
	Nodes []previewNode `json:"nodes"`
}

func validateBindings(d *gorm.DB, bindings []resourceBinding) ([]db.GroupSubscription, []previewSubscription, error) {
	if len(bindings) > 1000 {
		return nil, nil, fmt.Errorf("too many subscriptions")
	}
	result := []db.GroupSubscription{}
	preview := []previewSubscription{}
	seen := map[uint]bool{}
	for _, binding := range bindings {
		id, err := cursorID(binding.ID)
		if err != nil || seen[id] {
			return nil, nil, fmt.Errorf("invalid or duplicate subscription ID")
		}
		seen[id] = true
		var sub db.Subscription
		if err = d.First(&sub, id).Error; err != nil {
			return nil, nil, fmt.Errorf("subscription no longer exists; reopen the editor")
		}
		var filter *regexp.Regexp
		var normalized *string
		if binding.Regex != nil && strings.TrimSpace(*binding.Regex) != "" {
			value := strings.TrimSpace(*binding.Regex)
			if len(value) > 4096 {
				return nil, nil, fmt.Errorf("regex too long")
			}
			filter, err = regexp.Compile(value)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid regex (Go/RE2; lookaround and backreferences are unsupported): %w", err)
			}
			normalized = &value
		}
		var nodes []db.Node
		if err = d.Where("subscription_id = ?", id).Order("id").Find(&nodes).Error; err != nil {
			return nil, nil, err
		}
		item := previewSubscription{ID: binding.ID, Nodes: []previewNode{}}
		for _, node := range nodes {
			if filter == nil || filter.MatchString(node.Name) {
				item.Nodes = append(item.Nodes, previewNode{string(common.EncodeCursor(node.ID)), node.Name, node.Protocol})
			}
		}
		result = append(result, db.GroupSubscription{SubscriptionID: id, NameFilterRegex: normalized})
		preview = append(preview, item)
	}
	return result, preview, nil
}
func groupResourcesAPI(w http.ResponseWriter, r *http.Request) {
	d := db.DB(r.Context())
	if r.Method == "GET" && r.URL.Path == "/api/groups/resources" {
		id, err := cursorID(r.URL.Query().Get("id"))
		if err != nil {
			fail(w, 400, "invalid group ID")
			return
		}
		result, err := readGroupResources(d, id)
		if err != nil {
			fail(w, 404, "group no longer exists")
			return
		}
		reply(w, 200, result)
		return
	}
	if r.Method != "POST" {
		fail(w, 405, "method not allowed")
		return
	}
	var request groupResources
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil {
		fail(w, 400, "invalid group resource request")
		return
	}
	if r.URL.Path == "/api/groups/preview" {
		_, items, err := validateBindings(d, request.Subscriptions)
		if err != nil {
			fail(w, 400, err.Error())
			return
		}
		reply(w, 200, map[string]any{"items": items})
		return
	}
	id, err := cursorID(request.ID)
	if err != nil || (request.Mode != "nodes" && request.Mode != "subscriptions") {
		fail(w, 400, "invalid group or resource type")
		return
	}
	status := 400
	err = d.Transaction(func(tx *gorm.DB) error {
		current, err := readGroupResources(tx, id)
		if err != nil {
			status = 404
			return fmt.Errorf("group no longer exists")
		}
		if request.Revision != current.Revision {
			status = 409
			return fmt.Errorf("群组已在其他页面修改，请关闭后重新打开编辑框")
		}
		if request.Mode == "subscriptions" {
			bindings, _, err := validateBindings(tx, request.Subscriptions)
			if err != nil {
				return err
			}
			if err = tx.Where("group_id = ?", id).Delete(&db.GroupSubscription{}).Error; err != nil {
				return err
			}
			for i := range bindings {
				bindings[i].GroupID = id
			}
			if len(bindings) > 0 {
				if err = tx.Create(&bindings).Error; err != nil {
					return err
				}
			}
		} else {
			if len(request.NodeIDs) > 5000 {
				return fmt.Errorf("too many nodes")
			}
			nodes := []db.Node{}
			seen := map[uint]bool{}
			for _, value := range request.NodeIDs {
				nodeID, err := cursorID(value)
				if err != nil || seen[nodeID] {
					return fmt.Errorf("invalid or duplicate node ID")
				}
				seen[nodeID] = true
				var node db.Node
				if err = tx.First(&node, nodeID).Error; err != nil {
					return fmt.Errorf("node no longer exists; reopen the editor")
				}
				nodes = append(nodes, node)
			}
			if err = tx.Model(&db.Group{ID: id}).Association("Node").Replace(nodes); err != nil {
				return err
			}
		}
		return tx.Model(&db.Group{ID: id}).Update("version", gorm.Expr("version + 1")).Error
	})
	if err != nil {
		fail(w, status, err.Error())
		return
	}
	reply(w, 200, map[string]bool{"saved": true})
}
