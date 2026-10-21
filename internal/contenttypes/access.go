package contenttypes

import (
	"context"
	"errors"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// Access limits who sees which records of a type. A record belongs to the
// user whose e-mail address its Owner text field holds. Users without the All
// permission read, change and delete only their own records and are recorded
// as the owner of what they create; holders of All (admins always) see
// everything. Public reads are not affected: only the admin API knows users.
type Access struct {
	Owner string `json:"owner" yaml:"owner"`
	All   string `json:"all" yaml:"all"`
}

var errNotOwner = errors.New("record belongs to someone else")

// restriction reports whether the caller is limited to their own records
// and, if so, the owner value that identifies them.
func (a AdminAPI) restriction(r *http.Request, definition Definition) (string, bool) {
	if definition.Access == nil || (a.Can != nil && a.Can(r, definition.Access.All)) {
		return "", false
	}
	if a.Subject == nil {
		return "", true
	}
	return a.Subject(r), true
}

// owns reports whether the stored record belongs to subject. A caller with
// no identity owns nothing.
func (a AdminAPI) owns(ctx context.Context, definition Definition, subject, id string) bool {
	if subject == "" {
		return false
	}
	record, err := a.Service.GetRecord(ctx, definition.ID, id)
	if err != nil {
		return false
	}
	owner, _ := record.Data[definition.Access.Owner].(string)
	return owner == subject
}

// scopePayload applies the restriction to a write: the caller may only touch
// their own records and always ends up as the owner.
func (a AdminAPI) scopePayload(r *http.Request, definition Definition, id string, payload map[string]any) error {
	subject, restricted := a.restriction(r, definition)
	if !restricted {
		return nil
	}
	if subject == "" || (id != "" && !a.owns(r.Context(), definition, subject, id)) {
		if id == "" {
			return errBatchForbidden
		}
		return collections.ErrRecordNotFound
	}
	payload[definition.Access.Owner] = subject
	return nil
}

func (a AdminAPI) visibleTo(r *http.Request) func(Definition, collections.Record) bool {
	return func(definition Definition, record collections.Record) bool {
		subject, restricted := a.restriction(r, definition)
		if !restricted {
			return true
		}
		owner, _ := record.Data[definition.Access.Owner].(string)
		return subject != "" && owner == subject
	}
}
