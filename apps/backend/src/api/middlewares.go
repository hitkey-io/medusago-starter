// Package api holds the API routes and middlewares of the project.
package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
)

// Middlewares are registered in main.go. The product index declares
// filterable status and sales_channel_ids, so the route narrows it to
// published products in the key's sales channels.
var Middlewares = []medusahttp.MiddlewareRoute{
	{
		Methods: []string{"POST"},
		Matcher: "/store/search",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.ConfigureStoreSearch(middlewares.StoreSearchConfig{
				AllowedIndexes: map[string]any{"product": true},
			}),
		},
	},
}
