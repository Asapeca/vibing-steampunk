package adt

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// A change spread over the parts of one class (a local class declared in
// the local definitions, implemented in the local implementations; a method
// of the global class using a local type) is red when each part is checked
// alone: SAP compiles the part against the saved text of the others. The
// class is one check object, and its parts are artifacts of it, so they go
// to the check run together, each with its unsaved text.

// CheckArtifact is one unsaved part of an object for SyntaxCheckParts.
type CheckArtifact struct {
	URI     string `json:"uri"`
	Content string `json:"content"`
}

var classURLRe = regexp.MustCompile(`^/sap/bc/adt/oo/classes/[^/?#]+$`)

// classParts are the parts of a class a check run takes as artifacts.
var classParts = map[string]bool{
	"/source/main": true, "/includes/definitions": true, "/includes/implementations": true,
	"/includes/macros": true, "/includes/testclasses": true,
}

// SyntaxCheckParts checks several unsaved parts of one class in one check
// run. classURL is the class (/sap/bc/adt/oo/classes/zcl_foo); each part's
// URI is the class itself or its source/main for the main source, or one of
// its includes (…/includes/definitions, implementations, macros,
// testclasses). Nothing is saved.
func (c *Client) SyntaxCheckParts(ctx context.Context, classURL string, parts []CheckArtifact) ([]SyntaxCheckResult, error) {
	body, err := classCheckRunBody(classURL, parts)
	if err != nil {
		return nil, err
	}
	resp, err := c.transport.Request(ctx, "/sap/bc/adt/checkruns?reporters=abapCheckRun", &RequestOptions{
		Method:      http.MethodPost,
		Body:        []byte(body),
		ContentType: "application/*",
	})
	if err != nil {
		return nil, fmt.Errorf("syntax check failed: %w", err)
	}
	return parseSyntaxCheckResults(resp.Body)
}

// classCheckRunBody is the check run of the class with one artifact per part.
func classCheckRunBody(classURL string, parts []CheckArtifact) (string, error) {
	classURL = strings.TrimSuffix(classURL, "/")
	if !classURLRe.MatchString(classURL) {
		return "", fmt.Errorf("artifacts: object_url must be a class (/sap/bc/adt/oo/classes/<name>), got %q", classURL)
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("artifacts: at least one part is required")
	}
	seen := map[string]bool{}
	var b strings.Builder
	for _, p := range parts {
		uri := strings.TrimSuffix(p.URI, "/")
		if strings.EqualFold(uri, classURL) {
			uri = classURL + "/source/main"
		}
		if len(uri) <= len(classURL) || !strings.EqualFold(uri[:len(classURL)], classURL) ||
			!classParts[strings.ToLower(uri[len(classURL):])] {
			return "", fmt.Errorf("artifacts: %q is not a part of %s (source/main or includes/definitions, "+
				"implementations, macros, testclasses)", p.URI, classURL)
		}
		uri = classURL + strings.ToLower(uri[len(classURL):])
		if seen[uri] {
			return "", fmt.Errorf("artifacts: %s given twice", uri)
		}
		seen[uri] = true
		if p.Content == "" {
			return "", fmt.Errorf("artifacts: %s has no content", uri)
		}
		fmt.Fprintf(&b, `
      <chkrun:artifact chkrun:contentType="text/plain; charset=utf-8" chkrun:uri="%s">
        <chkrun:content>%s</chkrun:content>
      </chkrun:artifact>`, uri, base64.StdEncoding.EncodeToString([]byte(p.Content)))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<chkrun:checkObjectList xmlns:chkrun="http://www.sap.com/adt/checkrun" xmlns:adtcore="http://www.sap.com/adt/core">
  <chkrun:checkObject adtcore:uri="%s" chkrun:version="active">
    <chkrun:artifacts>%s
    </chkrun:artifacts>
  </chkrun:checkObject>
</chkrun:checkObjectList>`, classURL, b.String()), nil
}
