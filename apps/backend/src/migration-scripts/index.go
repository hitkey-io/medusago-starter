// Package migrationscripts holds the data migration scripts of the project:
// db:migrate (and db:setup) runs each of them once, after the migrations and
// the scripts of MedusaGo, in name order.
package migrationscripts

import (
	typescommon "github.com/hitkey-io/medusago/packages/core/types/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Scripts are registered in main.go. The key is the name recorded in the
// script_migrations table. It keeps the file name of the scripts of the
// Medusa starter, so a database that Medusa already seeded isn't seeded again.
var Scripts = map[string]typescommon.ExecScript{
	"initial-data-seed.ts":     InitialDataSeed,
	"region-store-policies.ts": RegionStorePolicies,
}

// rec builds a record from key-value pairs, keeping their order.
func rec(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

// records returns the records of a list field.
func records(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, e := range t {
			if r, ok := e.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// id returns the id of a record.
func id(r *dal.Record) string {
	v, _ := r.Get("id")
	s, _ := v.(string)
	return s
}
