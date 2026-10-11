package urltemplate

import "testing"

func TestParseRulesBySegmentCount(t *testing.T) {
	rules := ParseRulesBySegmentCount([]string{"/users/{id}", "/health", "/a/b/c"})
	if len(rules[1]) != 1 {
		t.Fatalf("segment count 1: got %d rules, want 1", len(rules[1]))
	}
	if len(rules[2]) != 1 {
		t.Fatalf("segment count 2: got %d rules, want 1", len(rules[2]))
	}
	if len(rules[3]) != 1 {
		t.Fatalf("segment count 3: got %d rules, want 1", len(rules[3]))
	}
}

func TestPathMatchesAnyRule(t *testing.T) {
	rules := ParseRulesBySegmentCount([]string{"/users/{id}", "/health"})

	if !PathMatchesAnyRule("/users/123", rules) {
		t.Fatal("expected /users/123 to match")
	}
	if !PathMatchesAnyRule("/health", rules) {
		t.Fatal("expected /health to match")
	}
	if PathMatchesAnyRule("/users/456/orders", rules) {
		t.Fatal("expected /users/456/orders not to match (different segment count)")
	}
	if PathMatchesAnyRule("/api/v1/items", rules) {
		t.Fatal("expected /api/v1/items not to match")
	}
	if PathMatchesAnyRule("/users/123", nil) {
		t.Fatal("nil rules should not match")
	}
}
