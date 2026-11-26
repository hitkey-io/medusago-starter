package helpers

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/search/model"
)

// PriceCurrencies are the currencies the index holds prices in. Each gets
// its own set of price fields, so filtering and sorting work per currency.
// Adding one is a schema change: the module reindexes on the next boot.
// Keep in sync with the storefront's SEARCH_PRICE_CURRENCIES.
var PriceCurrencies = []string{"eur", "usd"}

// Read once per currency, since the Pricing Module calculates for a single
// currency per query.
var priceGraphFields = []string{
	"id",
	"variants.calculated_price.calculated_amount",
	"variants.calculated_price.original_amount",
}

func pricingContext(currency string) *dal.Record {
	price := dal.NewRecord()
	price.Set("currency_code", currency)
	variants := dal.NewRecord()
	variants.Set("calculated_price", query.QueryContext(price))
	out := dal.NewRecord()
	out.Set("variants", variants)
	return out
}

// PriceFields are the per-currency price fields, to add to the index schema.
func PriceFields() []model.F {
	var fields []model.F
	for _, currency := range PriceCurrencies {
		fields = append(fields,
			model.Field("min_price_"+currency, model.Float().Filterable().Sortable().FacetableTypes("stats").Retrievable()),
			model.Field("max_price_"+currency, model.Float().Filterable().Sortable().FacetableTypes("stats").Retrievable()),
			model.Field("original_price_"+currency, model.Float().Retrievable()),
			model.Field("on_sale_"+currency, model.Boolean().Filterable().Facetable().Retrievable()),
		)
	}
	return fields
}

// ProductPricingRows are a product's priced variants per currency, as
// LoadPricing reads them.
type ProductPricingRows map[string][]*dal.Record

// setPricing writes one currency's price fields: the cheapest variant's
// calculated and original price, and the most expensive variant's as the
// max. Picking one variant for the pair keeps the discount describing a real
// product rather than mixing two variants' amounts. A product without a
// price in the currency writes nothing, so it drops out of that currency's
// price filter and sort.
func setPricing(document *dal.Record, currency string, variants []*dal.Record) {
	var cheapest *struct{ calculated, original float64 }
	var maxPrice *float64

	for _, variant := range variants {
		price, _ := Get(variant, "calculated_price").(*dal.Record)
		calculated, ok := Get(price, "calculated_amount").(float64)
		if !ok {
			continue
		}
		original, ok := Get(price, "original_amount").(float64)
		if !ok {
			original = calculated
		}
		if maxPrice == nil || calculated > *maxPrice {
			v := calculated
			maxPrice = &v
		}
		if cheapest == nil || calculated < cheapest.calculated {
			cheapest = &struct{ calculated, original float64 }{calculated, original}
		}
	}

	if cheapest == nil {
		return
	}
	document.Set("min_price_"+currency, cheapest.calculated)
	document.Set("max_price_"+currency, *maxPrice)
	document.Set("original_price_"+currency, cheapest.original)
	// Written even when false: "not on sale" is a real facet bucket, unlike
	// a missing value.
	document.Set("on_sale_"+currency, cheapest.original > cheapest.calculated)
}

// SetProductPricing writes every currency's price fields of one product.
func SetProductPricing(document *dal.Record, rows ProductPricingRows) {
	for _, currency := range PriceCurrencies {
		setPricing(document, currency, rows[currency])
	}
}

// LoadPricing reads the priced variants of the given products, per
// currency: one query.graph read per currency, however many products there
// are.
func LoadPricing(ctx context.Context, ids []string, ic searchtypes.IngestionContext) (map[string]ProductPricingRows, error) {
	pricing := map[string]ProductPricingRows{}
	if len(ids) == 0 {
		return pricing, nil
	}
	q, ok := ic.Container.Query.(*query.Query)
	if !ok {
		return nil, fmt.Errorf("query is not available")
	}
	filterIDs := make([]any, len(ids))
	for i, id := range ids {
		filterIDs[i] = id
	}
	for _, currency := range PriceCurrencies {
		filters := dal.NewRecord()
		filters.Set("id", filterIDs)
		result, err := q.Graph(ctx, query.GraphInput{
			Entity:  "product",
			Fields:  priceGraphFields,
			Filters: filters,
			Context: pricingContext(currency),
		}, joiner.Options{})
		if err != nil {
			return nil, err
		}
		for _, product := range Records(result.Data) {
			id, _ := Get(product, "id").(string)
			rows := pricing[id]
			if rows == nil {
				rows = ProductPricingRows{}
				pricing[id] = rows
			}
			rows[currency] = Records(Get(product, "variants"))
		}
	}
	return pricing, nil
}
