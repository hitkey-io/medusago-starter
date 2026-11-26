// Package helpers holds the helpers of the product search index.
package helpers

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Records returns the records of a query.graph relation (a list of rows).
func Records(v any) []*dal.Record {
	var out []*dal.Record
	switch t := v.(type) {
	case []any:
		for _, e := range t {
			if r, ok := e.(*dal.Record); ok {
				out = append(out, r)
			}
		}
	case []*dal.Record:
		out = t
	}
	return out
}

// Get returns a field of a row (nil when the row or the field is missing).
func Get(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

// Trimmed returns a string field without surrounding spaces, and whether it
// is a non-empty string.
func Trimmed(r *dal.Record, key string) (string, bool) {
	s, ok := Get(r, key).(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	return s, s != ""
}
