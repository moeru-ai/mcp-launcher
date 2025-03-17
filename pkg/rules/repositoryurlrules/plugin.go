package repositoryurlrules

import (
	"context"
	"sort"

	"github.com/moeru-ai/mcp-launcher/pkg/plugins"
)

// RulesPlugin provides base functionality for rule-based plugins
type RulesPlugin struct {
	plugins.IsPlugin

	repositoryURLRules []Rule
}

func Rules(rules ...Rule) RulesPlugin {
	// Sort rules by priority
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority() > rules[j].Priority()
	})

	return RulesPlugin{repositoryURLRules: rules}
}

func (p *RulesPlugin) ShouldHandle(ctx context.Context, repoURL string) bool {
	for _, rule := range p.repositoryURLRules {
		if rule.MatchRepositoryURL(ctx, repoURL) {
			return true
		}
	}

	return false
}
