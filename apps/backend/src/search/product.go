// Package search declares the search indexes of the project. Each
// definition registers itself when the package is imported (main.go imports
// it), and the Search Module creates, seeds and updates the indexes.
package search

import (
	"context"

	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	searchutils "github.com/hitkey-io/medusago/packages/core/utils/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/search/model"

	"backend/src/search/helpers"
)

var productGraphFields = []string{
	"id",
	"title",
	"description",
	"handle",
	"thumbnail",
	"status",
	"created_at",
	"sales_channels.id",
	"categories.name",
	"tags.value",
	"options.title",
	"options.values.value",
}

var productFields = model.Define(append([]model.F{
	model.Field("id", model.Keyword().Filterable().Retrievable()),
	// Read by /store/search to scope the index; never returned to hits.
	model.Field("status", model.Keyword().Filterable().Retrievable(false)),
	model.Field("sales_channel_ids", model.Keyword().Array().Filterable().Retrievable(false)),
	model.Field("title", model.Text().SearchableWeight(3).Sortable().Retrievable()),
	model.Field("description", model.Text().SearchableWeight(1)),
	model.Field("handle", model.Keyword().Retrievable()),
	// Returned on hits only: never filtered, sorted or faceted on.
	model.Field("thumbnail", model.Keyword().Retrievable()),
	model.Field("created_at", model.Date().Sortable().Retrievable()),
	model.Field("category", model.Keyword().Array().Filterable().Facetable().Retrievable()),
	model.Field("labels", model.Keyword().Array().Filterable().Facetable().Retrievable()),
	model.Field("option_values", model.Keyword().Array().SearchableWeight(2).Filterable().Facetable().Retrievable()),
}, helpers.PriceFields()...)...)

// trimmedValues returns the non-empty values of a field of related rows.
func trimmedValues(rows any, key string) []any {
	out := []any{}
	for _, row := range helpers.Records(rows) {
		if v, ok := helpers.Trimmed(row, key); ok {
			out = append(out, v)
		}
	}
	return out
}

func toDocument(product *dal.Record, pricing helpers.ProductPricingRows) *dal.Record {
	d := dal.NewRecord()
	d.Set("id", helpers.Get(product, "id"))
	d.Set("status", helpers.Get(product, "status"))
	d.Set("sales_channel_ids", trimmedValues(helpers.Get(product, "sales_channels"), "id"))
	d.Set("title", helpers.Get(product, "title"))
	d.Set("description", helpers.Get(product, "description"))
	d.Set("handle", helpers.Get(product, "handle"))
	d.Set("thumbnail", helpers.Get(product, "thumbnail"))
	d.Set("created_at", helpers.Get(product, "created_at"))
	d.Set("category", trimmedValues(helpers.Get(product, "categories"), "name"))
	d.Set("labels", trimmedValues(helpers.Get(product, "tags"), "value"))
	d.Set("option_values", helpers.ToOptionValues(helpers.Get(product, "options")))
	helpers.SetProductPricing(d, pricing)
	return d
}

// A row that returns no document leaves the index: the helpers turn it into
// a delete on consume and on the catch-up pass.
var source = searchutils.GraphSourceOptions{
	Fields: productGraphFields,
	Transform: func(ctx context.Context, rows []*dal.Record, ic searchtypes.IngestionContext) ([]*dal.Record, error) {
		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			id, _ := helpers.Get(row, "id").(string)
			ids = append(ids, id)
		}
		pricing, err := helpers.LoadPricing(ctx, ids, ic)
		if err != nil {
			return nil, err
		}
		documents := make([]*dal.Record, len(rows))
		for i, row := range rows {
			documents[i] = toDocument(row, pricing[ids[i]])
		}
		return documents, nil
	},
}

// productEvents is everything a document is built from, so a change to any
// of it re-indexes the products it touches. Only product.deleted removes
// documents: a deleted variant, option, tag or category means the product
// has to be read again.
//
// TODO: price lists emit no events (create, update, delete, batch prices).
// TODO: price lists crossing their start or end date change prices silently.
// TODO: linkProductsToSalesChannelWorkflow emits nothing.
// TODO: batchLinkProductsToCategoryWorkflow emits nothing.
// TODO: upsertVariantPricesWorkflow emits nothing when called directly.
var productEvents = []string{
	"product.created",
	"product.updated",
	"product.deleted",
	"product-variant.created",
	"product-variant.updated",
	"product-variant.deleted",
	"product-option.created",
	"product-option.updated",
	"product-option.deleted",
	"product-option-value.updated",
	"product-option-value.deleted",
	"product-tag.updated",
	"product-tag.deleted",
	"product-category.updated",
	"product-category.deleted",
	"sales-channel.deleted",
}

var typoToleranceEnabled = true

// Product is the "product" index the storefront searches and filters.
var Product = searchutils.DefineSearchIndex(searchtypes.Definition{
	Name:       "product",
	Entity:     "product",
	PrimaryKey: "id",
	Fields:     productFields,
	Settings: &searchtypes.Settings{
		TypoTolerance: &searchtypes.TypoTolerance{Enabled: &typoToleranceEnabled},
	},
	Events: productEvents,
	Consume: searchutils.GraphConsume(searchutils.GraphConsumeOptions{
		GraphSourceOptions: source,
		ResolveIDs:         helpers.ResolveProductIDs,
		IsDelete:           func(event searchtypes.Event) bool { return event.Name == "product.deleted" },
	}),
	Seed: searchutils.GraphSeed(searchutils.GraphSeedOptions{GraphSourceOptions: source}),
})
