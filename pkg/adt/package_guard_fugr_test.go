package adt

import "testing"

// WriteSource INCL of a function group's include: the gate names the URL to use.
func TestFunctionGroupIncludeURL(t *testing.T) {
	hits := []SearchResult{
		{URI: "/sap/bc/adt/functions/groups/zpp_sf_notes_fg/includes/lzpp_sf_notes_fgimp", Type: "FUGR/I",
			Name: "LZPP_SF_NOTES_FGIMP", PackageName: "ZPP_SFC"},
	}
	got := functionGroupIncludeURL("/sap/bc/adt/programs/includes/LZPP_SF_NOTES_FGIMP", "LZPP_SF_NOTES_FGIMP", hits)
	if got != hits[0].URI {
		t.Fatalf("got %q", got)
	}
	if got := functionGroupIncludeURL("/sap/bc/adt/programs/includes/ZOTHER", "ZOTHER", hits); got != "" {
		t.Fatalf("another name: %q", got)
	}
	if got := functionGroupIncludeURL("/sap/bc/adt/programs/programs/LZPP_SF_NOTES_FGIMP", "LZPP_SF_NOTES_FGIMP", hits); got != "" {
		t.Fatalf("not an include URL: %q", got)
	}
}
