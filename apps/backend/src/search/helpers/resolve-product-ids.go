package helpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

const resolveBatchSize = 200

// payloadIDs returns the ids of an event payload: core emits either a single
// { id } or a batch of them.
func payloadIDs(data any) []any {
	var entries []any
	switch t := data.(type) {
	case []any:
		entries = t
	case []*dal.Record:
		for _, r := range t {
			entries = append(entries, r)
		}
	default:
		entries = []any{data}
	}
	ids := []any{}
	for _, entry := range entries {
		var id any
		switch e := entry.(type) {
		case *dal.Record:
			id = Get(e, "id")
		case map[string]any:
			id = e["id"]
		}
		if s, ok := id.(string); ok && s != "" {
			ids = append(ids, s)
		}
	}
	return ids
}

// relatedProductIDs reads the products behind rows of another entity
// through query.graph. Deleted rows are soft-deleted, so they are still
// readable with WithDeleted, which is how a deleted variant or category
// still leads back to its products.
func relatedProductIDs(ctx context.Context, q *query.Query, entity string, fields []string, ids []any, pick func(row *dal.Record) []any, withDeleted bool) ([]any, error) {
	filters := dal.NewRecord()
	filters.Set("id", ids)
	result, err := q.Graph(ctx, query.GraphInput{Entity: entity, Fields: fields, Filters: filters, WithDeleted: withDeleted}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, row := range Records(result.Data) {
		for _, id := range pick(row) {
			if s, ok := id.(string); ok && s != "" {
				out = append(out, s)
			}
		}
	}
	return out, nil
}

// productIDsInSalesChannels searches the index: deleting a sales channel
// removes its product links, so the products can no longer be found through
// query.graph. The index still holds the channel on each document.
func productIDsInSalesChannels(ctx context.Context, q *query.Query, salesChannelIDs []any) ([]any, error) {
	ids := []any{}
	skip := 0
	for {
		filters := dal.NewRecord()
		filters.Set("sales_channel_ids", salesChannelIDs)
		pagination := dal.NewRecord()
		pagination.Set("skip", float64(skip))
		pagination.Set("take", float64(resolveBatchSize))
		request := dal.NewRecord()
		request.Set("entity", "product")
		request.Set("fields", []any{"id"})
		request.Set("filters", filters)
		request.Set("pagination", pagination)
		results, err := q.Search(ctx, []*dal.Record{request}, joiner.Options{})
		if err != nil {
			return nil, err
		}
		if len(results) == 0 {
			return ids, nil
		}
		hits, _ := Get(results[0].SearchResult, "hits").([]any)
		for _, hit := range hits {
			if r, ok := hit.(*dal.Record); ok {
				ids = append(ids, Get(r, "id"))
			}
		}
		if len(hits) < resolveBatchSize {
			return ids, nil
		}
		skip += resolveBatchSize
	}
}

// productsOf picks the ids of a row's "products" relation.
func productsOf(row *dal.Record) []any {
	var out []any
	for _, product := range Records(Get(row, "products")) {
		out = append(out, Get(product, "id"))
	}
	return out
}

// ResolveProductIDs returns the products an event affects. A product event
// carries them directly; an event about a variant, option, tag, category or
// sales channel is mapped to the products behind it.
func ResolveProductIDs(ctx context.Context, event searchtypes.Event, ic searchtypes.IngestionContext) (any, error) {
	ids := payloadIDs(event.Data)
	if len(ids) == 0 {
		return []any{}, nil
	}
	q, ok := ic.Container.Query.(*query.Query)
	if !ok {
		return nil, fmt.Errorf("query is not available")
	}
	entity, _, _ := strings.Cut(event.Name, ".")
	deleted := strings.HasSuffix(event.Name, ".deleted")
	productIDOf := func(row *dal.Record) []any { return []any{Get(row, "product_id")} }

	switch entity {
	case "product":
		return ids, nil
	case "product-variant":
		return relatedProductIDs(ctx, q, "product_variant", []string{"product_id"}, ids, productIDOf, deleted)
	case "product-option":
		return relatedProductIDs(ctx, q, "product_option", []string{"product_id"}, ids, productIDOf, deleted)
	case "product-option-value":
		return relatedProductIDs(ctx, q, "product_option_value", []string{"option.product_id"}, ids, func(row *dal.Record) []any {
			option, _ := Get(row, "option").(*dal.Record)
			return []any{Get(option, "product_id")}
		}, deleted)
	case "product-tag":
		return relatedProductIDs(ctx, q, "product_tag", []string{"products.id"}, ids, productsOf, deleted)
	case "product-category":
		return relatedProductIDs(ctx, q, "product_category", []string{"products.id"}, ids, productsOf, deleted)
	case "sales-channel":
		return productIDsInSalesChannels(ctx, q, ids)
	default:
		return []any{}, nil
	}
}
