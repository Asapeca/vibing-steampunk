package adt

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// checkRunServer answers mainprograms lookups with mains (none: 404) and
// records the body of every check run.
func checkRunServer(t *testing.T, mains string) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-csrf-token", "test-token")
		switch {
		case strings.HasSuffix(r.URL.Path, "/mainprograms"):
			if mains == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write([]byte(mains))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/checkruns"):
			b, _ := io.ReadAll(r.Body)
			mu.Lock()
			bodies = append(bodies, string(b))
			mu.Unlock()
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<chkrun:checkRunReports xmlns:chkrun="http://www.sap.com/adt/checkrun"/>`))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	return srv, func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), bodies...) }
}

// A program include is checked in the context of its main program: without
// one ADT answers no messages whatever the content (seen live on PDS 7.5x).
func TestSyntaxCheckGivesAProgramIncludeItsMainProgram(t *testing.T) {
	srv, bodies := checkRunServer(t, mainProgramXML("ZDEMO_MAIN", "ZDEMO_OTHER"))
	defer srv.Close()
	c := NewClient(srv.URL, "user", "pass")
	if _, err := c.SyntaxCheck(context.Background(), "/sap/bc/adt/programs/includes/zdemo_c01", "x."); err != nil {
		t.Fatalf("SyntaxCheck: %v", err)
	}
	got := bodies()
	want := `adtcore:uri="/sap/bc/adt/programs/includes/zdemo_c01?context=%2Fsap%2Fbc%2Fadt%2Fprograms%2Fprograms%2Fzdemo_main"`
	if len(got) != 1 || !strings.Contains(got[0], want) {
		t.Fatalf("check run without the main program's context: %v", got)
	}
	if !strings.Contains(got[0], `chkrun:uri="/sap/bc/adt/programs/includes/zdemo_c01/source/main"`) {
		t.Errorf("artifact uri changed: %s", got[0])
	}
}

// An include whose main program is not known is not checked: ADT would
// answer no messages whatever the text, a green that checked nothing.
func TestSyntaxCheckRefusesAnIncludeWithoutAMainProgram(t *testing.T) {
	for name, mains := range map[string]string{
		"lookup fails": "",
		"no programs":  mainProgramXML(),
	} {
		srv, bodies := checkRunServer(t, mains)
		_, err := NewClient(srv.URL, "user", "pass").SyntaxCheck(context.Background(),
			"/sap/bc/adt/programs/includes/zdemo_c01", "x.")
		if err == nil || !strings.Contains(err.Error(), "not checked") {
			t.Errorf("%s: want the check refused, got %v", name, err)
		}
		if got := bodies(); len(got) != 0 {
			t.Errorf("%s: a check run without a context was sent: %v", name, got)
		}
		srv.Close()
	}
}

// A program and a class are checked as before.
func TestSyntaxCheckLeavesOtherObjectsAsTheyWere(t *testing.T) {
	for name, tc := range map[string]struct{ url, mains string }{
		"program": {"/sap/bc/adt/programs/programs/zdemo", mainProgramXML("ZDEMO_MAIN")},
		"class":   {"/sap/bc/adt/oo/classes/zcl_demo", mainProgramXML("ZDEMO_MAIN")},
	} {
		srv, bodies := checkRunServer(t, tc.mains)
		if _, err := NewClient(srv.URL, "user", "pass").SyntaxCheck(context.Background(), tc.url, "x."); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := bodies(); len(got) != 1 || strings.Contains(got[0], "context=") ||
			!strings.Contains(got[0], `adtcore:uri="`+tc.url+`"`) {
			t.Errorf("%s: %v", name, got)
		}
		srv.Close()
	}
}
