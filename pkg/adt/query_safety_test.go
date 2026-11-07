package adt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// GetTableContents is a predefined query (Q): a safety config that excludes
// Q refuses it before anything is sent, with or without a statement.
func TestGetTableContentsHonoursQuery(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	for _, tc := range []struct {
		name   string
		safety SafetyConfig
	}{
		{"disallowed Q", SafetyConfig{DisallowedOps: "Q"}},
		{"allowlist without Q", SafetyConfig{AllowedOps: "RS"}},
		{"read-only with Q disallowed", SafetyConfig{ReadOnly: true, BlockFreeSQL: true, DisallowedOps: "Q"}},
	} {
		c := NewClient(srv.URL, "u", "p", WithSafety(tc.safety))
		for _, filter := range []string{"", "SELECT * FROM T000"} {
			if _, err := c.GetTableContents(context.Background(), "T000", 10, filter); err == nil {
				t.Fatalf("%s, filter %q: want refused", tc.name, filter)
			}
		}
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("refused reads must not reach SAP, got %d requests", n)
	}

	// Q allowed: the plain read goes out (the fake server's 500 is fine here).
	c := NewClient(srv.URL, "u", "p", WithSafety(SafetyConfig{ReadOnly: true, BlockFreeSQL: true}))
	_, _ = c.GetTableContents(context.Background(), "T000", 10, "")
	if hits.Load() == 0 {
		t.Fatal("with Q allowed the read must be sent")
	}
}
