package adt

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
)

func TestSyntaxCheckPartsSendsOneCheckObjectWithEveryPart(t *testing.T) {
	srv, bodies := checkRunServer(t, "")
	defer srv.Close()
	c := NewClient(srv.URL, "u", "p")
	_, err := c.SyntaxCheckParts(context.Background(), "/sap/bc/adt/oo/classes/zcl_x", []CheckArtifact{
		{URI: "/sap/bc/adt/oo/classes/zcl_x/includes/definitions", Content: "CLASS lcl DEFINITION. ENDCLASS."},
		{URI: "/sap/bc/adt/oo/classes/zcl_x/includes/IMPLEMENTATIONS", Content: "CLASS lcl IMPLEMENTATION. ENDCLASS."},
		{URI: "/sap/bc/adt/oo/classes/zcl_x", Content: "CLASS zcl_x DEFINITION PUBLIC."},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := bodies()
	if len(got) != 1 {
		t.Fatalf("check runs: %d, want 1", len(got))
	}
	b := got[0]
	if strings.Count(b, "<chkrun:checkObject ") != 1 || !strings.Contains(b, `adtcore:uri="/sap/bc/adt/oo/classes/zcl_x"`) {
		t.Errorf("one check object of the class expected:\n%s", b)
	}
	for _, want := range []string{
		`chkrun:uri="/sap/bc/adt/oo/classes/zcl_x/includes/definitions"`,
		`chkrun:uri="/sap/bc/adt/oo/classes/zcl_x/includes/implementations"`,
		`chkrun:uri="/sap/bc/adt/oo/classes/zcl_x/source/main"`,
		base64.StdEncoding.EncodeToString([]byte("CLASS lcl IMPLEMENTATION. ENDCLASS.")),
	} {
		if !strings.Contains(b, want) {
			t.Errorf("body lacks %s:\n%s", want, b)
		}
	}
}

func TestSyntaxCheckPartsRefusesForeignParts(t *testing.T) {
	for _, tc := range []struct {
		url   string
		parts []CheckArtifact
	}{
		{"/sap/bc/adt/programs/programs/zx", []CheckArtifact{{URI: "/sap/bc/adt/programs/programs/zx/source/main", Content: "x"}}},
		{"/sap/bc/adt/oo/classes/zcl_x", nil},
		{"/sap/bc/adt/oo/classes/zcl_x", []CheckArtifact{{URI: "/sap/bc/adt/oo/classes/zcl_y/includes/definitions", Content: "x"}}},
		{"/sap/bc/adt/oo/classes/zcl_x", []CheckArtifact{{URI: "/sap/bc/adt/oo/classes/zcl_x/includes/other", Content: "x"}}},
		{"/sap/bc/adt/oo/classes/zcl_x", []CheckArtifact{{URI: "/sap/bc/adt/oo/classes/zcl_x/includes/definitions", Content: ""}}},
		{"/sap/bc/adt/oo/classes/zcl_x", []CheckArtifact{
			{URI: "/sap/bc/adt/oo/classes/zcl_x", Content: "a"}, {URI: "/sap/bc/adt/oo/classes/zcl_x/source/main", Content: "b"}}},
	} {
		if _, err := classCheckRunBody(tc.url, tc.parts); err == nil {
			t.Errorf("%s %v: no error", tc.url, tc.parts)
		}
	}
}
