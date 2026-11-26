// Package lib holds values shared by the customizations of the project.
package lib

import "github.com/hitkey-io/medusago/packages/core/utils/src/dal"

// DefaultRegionStorePolicies returns the store policy values kept in each
// region's metadata. The storefront renders the shipping and returns pages
// and the structured data from these keys, and merchants can change them in
// Settings → Regions → Metadata.
func DefaultRegionStorePolicies() *dal.Record {
	r := dal.NewRecord()
	r.Set("return_window_days", float64(30))
	r.Set("free_returns", false)
	r.Set("shipping_rate", float64(10))
	r.Set("handling_days_min", float64(0))
	r.Set("handling_days_max", float64(1))
	r.Set("transit_days_min", float64(2))
	r.Set("transit_days_max", float64(3))
	return r
}
