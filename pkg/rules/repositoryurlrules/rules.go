package repositoryurlrules

import (
	"context"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/nekomeowww/fo"
)

type Rule interface {
	MatchRepositoryURL(ctx context.Context, repoURL string) bool
	Priority() int
}

var _ Rule = (*ExactMatchRule)(nil)

// ExactMatchRule matches exact repository URLs
type ExactMatchRule struct {
	url string
}

func ForExact(url string) *ExactMatchRule {
	return &ExactMatchRule{url: url}
}

func (r *ExactMatchRule) MatchRepositoryURL(ctx context.Context, repoURL string) bool {
	return r.url == repoURL
}

// Highest priority
func (r *ExactMatchRule) Priority() int {
	return 100 //nolint:mnd
}

var _ Rule = (*PatternMatchRule)(nil)

// PatternMatchRule matches using glob patterns
type PatternMatchRule struct {
	pattern string
}

func ForPattern(pattern string) *PatternMatchRule {
	return &PatternMatchRule{pattern: pattern}
}

func (r *PatternMatchRule) MatchRepositoryURL(ctx context.Context, repoURL string) bool {
	return fo.May(doublestar.Match(r.pattern, repoURL))
}

// Medium priority
func (r *PatternMatchRule) Priority() int {
	return 50 //nolint:mnd
}
