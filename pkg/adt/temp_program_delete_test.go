package adt

import (
	"context"
	"strings"
	"testing"
)

// Deletion refused: the shape of a deployment that lets an agent write but
// never delete (--disallowed-ops D, writes limited to its packages and $TMP).
var noDeleteSafety = SafetyConfig{DisallowedOps: "D", AllowedPackages: []string{"$TMP", "ZPROJECT"}}

func TestExecuteABAPDeletesItsOwnProgramWhereDeleteIsRefused(t *testing.T) {
	srv := &executeCleanupServer{}
	result, err := srv.start(t, WithSafety(noDeleteSafety)).ExecuteABAP(context.Background(), "lv_result = 'ok'.", nil)
	if err != nil {
		t.Fatalf("ExecuteABAP: %v", err)
	}
	if !result.CleanedUp || srv.count("delete") != 1 {
		t.Fatalf("own temporary program not deleted: cleaned=%v deletes=%d %q", result.CleanedUp, srv.count("delete"), result.Message)
	}
}

func TestCheckABAPIsNotRefusedUpFrontWhereDeleteIsRefused(t *testing.T) {
	srv := &executeCleanupServer{}
	_, err := srv.start(t, WithSafety(noDeleteSafety)).CheckABAP(context.Background(), "WRITE 'x'.")
	if err != nil && strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("CheckABAP refused up front: %v", err)
	}
	if srv.count("create") != 1 || srv.count("delete") != 1 {
		t.Fatalf("create=%d delete=%d, want 1 and 1", srv.count("create"), srv.count("delete"))
	}
}

func TestOtherDeletesStayRefused(t *testing.T) {
	srv := &executeCleanupServer{}
	c := srv.start(t, WithSafety(noDeleteSafety))
	ctx := context.Background()
	own := "/sap/bc/adt/programs/programs/ZTEMP_EXEC_12345678"
	other := "/sap/bc/adt/programs/programs/ZPROJECT_REPORT"

	// A plain delete, of any object.
	if err := c.DeleteObject(ctx, other, "H", ""); err == nil {
		t.Error("DeleteObject of a project object passed")
	}
	// A plain delete of a program this client did create: the mark is missing.
	c.ownTemps.add(own)
	if err := c.DeleteObject(ctx, own, "H", ""); err == nil {
		t.Error("DeleteObject of an own temporary program without the cleanup mark passed")
	}
	// The cleanup path, for an object this client did not create.
	if w := c.deleteTemporaryProgram(ctx, other, "ZPROJECT_REPORT"); len(w) == 0 {
		t.Error("cleanup deleted an object it did not create")
	}
	// A marked context forged around a plain DeleteObject of another object.
	if err := c.DeleteObject(withOwnTemporaryDelete(ctx, own), other, "H", ""); err == nil {
		t.Error("a mark for one object let another be deleted")
	}
	// Once deleted, the program is no longer own: a second cleanup is refused.
	// (The workflows mark the package as checked when they create the program.)
	ctx = withMutationPackageChecked(ctx, own)
	if w := c.deleteTemporaryProgram(ctx, own, "ZTEMP_EXEC_12345678"); len(w) != 0 {
		t.Fatalf("first cleanup of an own program: %v", w)
	}
	if w := c.deleteTemporaryProgram(ctx, own, "ZTEMP_EXEC_12345678"); len(w) == 0 {
		t.Error("an own program was deleted twice")
	}
	if got := srv.count("delete"); got != 1 {
		t.Fatalf("DELETE requests = %d, want exactly the one own cleanup", got)
	}
}

func TestOwnTemporaryDeleteNeverUnderReadOnly(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "u", "p", WithSafety(SafetyConfig{ReadOnly: true}))
	own := "/sap/bc/adt/programs/programs/ZTEMP_EXEC_1"
	c.ownTemps.add(own)
	m := MutationContext{Op: OpDelete, OpName: "DeleteObject", ObjectURL: own}
	if c.ownTemporaryDelete(withOwnTemporaryDelete(context.Background(), own), m) {
		t.Fatal("read-only must refuse every delete")
	}
}
