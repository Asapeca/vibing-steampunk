package adt

import (
	"context"
	"sync"
)

// Deleting a workflow's own temporary program.
//
// ExecuteABAP and CheckABAP create a program in $TMP and delete it again. With
// deletion refused (--disallowed-ops D, or an --allowed-ops list without D)
// that cleanup used to fail and leave the program behind, and CheckABAP was
// refused outright. A deployment that forbids deleting objects still wants the
// workflows that only delete what they created themselves.
//
// The exception is narrow. A DELETE passes the operation-type check despite D
// being refused only when all of these hold:
//   - it comes from the workflow's own cleanup (deleteTemporaryProgram marks
//     its context with the object's URL);
//   - this client created that very object in this process (CreateObject of
//     the temporary program succeeded and recorded it);
//   - the session is not --read-only.
//
// Everything else of the gate still applies to that DELETE: the package
// whitelist ($TMP must be allowed) and the transportable-edit check. Any other
// delete — DeleteObject, SAP(action="delete"), a rename's delete of the old
// object — is refused as before.

// ownTemporaryPrograms are the temporary programs this client created.
type ownTemporaryPrograms struct {
	mu   sync.Mutex
	urls map[string]struct{}
}

func (o *ownTemporaryPrograms) add(objectURL string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.urls == nil {
		o.urls = map[string]struct{}{}
	}
	o.urls[mutationPackageKey(objectURL)] = struct{}{}
}

func (o *ownTemporaryPrograms) remove(objectURL string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.urls, mutationPackageKey(objectURL))
}

func (o *ownTemporaryPrograms) has(objectURL string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	_, ok := o.urls[mutationPackageKey(objectURL)]
	return ok
}

type ownTemporaryDeleteKey struct{}

// withOwnTemporaryDelete marks ctx: the DELETE of objectURL that follows is a
// workflow removing its own temporary program. objectURL "" marks the up-front
// check of a workflow that has not created the program yet (CheckABAP).
func withOwnTemporaryDelete(ctx context.Context, objectURL string) context.Context {
	return context.WithValue(ctx, ownTemporaryDeleteKey{}, mutationPackageKey(objectURL))
}

// ownTemporaryDelete reports whether m is a DELETE covered by the exception.
func (c *Client) ownTemporaryDelete(ctx context.Context, m MutationContext) bool {
	if m.Op != OpDelete || c.config.Safety.ReadOnly {
		return false
	}
	marked, ok := ctx.Value(ownTemporaryDeleteKey{}).(string)
	if !ok {
		return false
	}
	if m.ObjectURL == "" { // up-front check: only for a program in $TMP not created yet
		return marked == "" && m.Package == "$TMP"
	}
	return marked != "" && marked == mutationPackageKey(m.ObjectURL) && c.ownTemps.has(m.ObjectURL)
}
