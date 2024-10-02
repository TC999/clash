package route

import (
	"net/http"

	"github.com/doreamon-design/clash/tunnel"
	"github.com/go-zoox/zoox"
)

type Rule struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
}

func getRules(ctx *zoox.Context) {
	rawRules := tunnel.Rules()

	rules := []Rule{}
	for _, rule := range rawRules {
		rules = append(rules, Rule{
			Type:    rule.RuleType().String(),
			Payload: rule.Payload(),
			Proxy:   rule.Adapter(),
		})
	}

	ctx.JSON(http.StatusOK, zoox.H{
		"rules": rules,
	})
}
