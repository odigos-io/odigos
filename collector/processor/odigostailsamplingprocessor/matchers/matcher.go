package matchers

import "go.opentelemetry.io/collector/pdata/ptrace"

type Matcher interface {
	Match(span ptrace.Span) bool
}

type anyMatcher struct{}

func (anyMatcher) Match(span ptrace.Span) bool {
	return true
}

type compositeMatcher struct {
	matchers []Matcher
}

func newCompositeMatcher(matchers ...Matcher) Matcher {
	return &compositeMatcher{matchers: matchers}
}

func (m *compositeMatcher) Match(span ptrace.Span) bool {
	for _, matcher := range m.matchers {
		if !matcher.Match(span) {
			return false
		}
	}
	return true
}
