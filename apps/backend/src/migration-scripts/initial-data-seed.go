package migrationscripts

import (
	"context"
	"fmt"
	"log/slog"

	apikeyworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/api-key/workflows"
	fulfillmentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	inventoryworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/workflows"
	productcategoryworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/product-category/workflows"
	productworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	regionworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/region/workflows"
	saleschannelworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/sales-channel/workflows"
	stocklocationworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/stock-location/workflows"
	storeworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/store/workflows"
	taxworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/tax/workflows"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	typescommon "github.com/hitkey-io/medusago/packages/core/types/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"

	"backend/src/lib"
)

// fulfillmentService is the part of the Fulfillment Module the seed uses.
type fulfillmentService interface {
	CreateFulfillmentSets(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
}

// imageURL returns the URL of a demo product image.
func imageURL(name string) *dal.Record {
	return rec("url", "https://medusa-public-images.s3.eu-west-1.amazonaws.com/"+name)
}

// prices are the prices of every demo variant.
func prices() []any {
	return []any{
		rec("amount", float64(10), "currency_code", "eur"),
		rec("amount", float64(15), "currency_code", "usd"),
	}
}

var sizes = []string{"S", "M", "L", "XL"}

// sizeVariants returns one variant per size, with SKUs such as "SHORTS-M".
func sizeVariants(skuPrefix string) []any {
	var variants []any
	for _, size := range sizes {
		variants = append(variants, rec(
			"title", size,
			"sku", skuPrefix+"-"+size,
			"options", rec("Size", size),
			"prices", prices(),
		))
	}
	return variants
}

// shirtVariants returns one variant per size and colour.
func shirtVariants() []any {
	var variants []any
	for _, size := range sizes {
		for _, color := range []string{"Black", "White"} {
			variants = append(variants, rec(
				"title", size+" / "+color,
				"sku", fmt.Sprintf("SHIRT-%s-%s", size, map[string]string{"Black": "BLACK", "White": "WHITE"}[color]),
				"options", rec("Size", size, "Color", color),
				"prices", prices(),
			))
		}
	}
	return variants
}

// InitialDataSeed creates the data of a new store: a sales channel and its
// publishable API key, the store, a region with its tax regions, a stock
// location with shipping options, product categories and options, four
// products and their inventory levels.
func InitialDataSeed(ctx context.Context, args typescommon.ExecArgs) error {
	c := args.Container
	logger := c.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
	link := c.Resolve(common.ContainerRegistrationKeys.LINK).(*modulessdk.Link)
	q := c.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
	fulfillmentModuleService := c.Resolve(utilsmodulessdk.FULFILLMENT).(fulfillmentService)

	countries := []any{"gb", "de", "dk", "se", "fr", "es", "it"}

	logger.Info("Seeding store data...")
	salesChannels, err := saleschannelworkflows.CreateSalesChannelsWorkflow.Run(ctx, c, saleschannelworkflows.CreateSalesChannelsInput{
		SalesChannelsData: []*dal.Record{
			rec("name", "Default Sales Channel", "description", "Created by Medusa"),
		},
	})
	if err != nil {
		return err
	}
	defaultSalesChannelID := id(salesChannels[0])

	apiKeys, err := apikeyworkflows.CreateApiKeysWorkflow.Run(ctx, c, []*dal.Record{
		rec("title", "Default Publishable API Key", "type", "publishable", "created_by", ""),
	})
	if err != nil {
		return err
	}

	if _, err := apikeyworkflows.LinkSalesChannelsToApiKeyWorkflow.Run(ctx, c, apikeyworkflows.LinkInput{
		ID:  id(apiKeys[0]),
		Add: []string{defaultSalesChannelID},
	}); err != nil {
		return err
	}

	if _, err := storeworkflows.CreateStoresWorkflow.Run(ctx, c, storeworkflows.CreateStoresWorkflowInput{
		Stores: []*dal.Record{rec(
			"name", "Default Store",
			"supported_currencies", []any{
				rec("currency_code", "eur", "is_default", true),
				rec("currency_code", "usd", "is_default", false),
			},
			"default_sales_channel_id", defaultSalesChannelID,
		)},
	}); err != nil {
		return err
	}

	logger.Info("Seeding region data...")
	regions, err := regionworkflows.CreateRegionsWorkflow.Run(ctx, c, regionworkflows.CreateRegionsInput{
		Regions: []*dal.Record{rec(
			"name", "Europe",
			"currency_code", "eur",
			"countries", countries,
			"payment_providers", []any{"pp_system_default"},
			"metadata", lib.DefaultRegionStorePolicies(),
		)},
	})
	if err != nil {
		return err
	}
	regionID := id(regions[0])
	logger.Info("Finished seeding regions.")

	logger.Info("Seeding tax regions...")
	taxRegions := make([]*dal.Record, len(countries))
	for i, country := range countries {
		taxRegions[i] = rec("country_code", country, "provider_id", "tp_system")
	}
	if _, err := taxworkflows.CreateTaxRegionsWorkflow.Run(ctx, c, taxRegions); err != nil {
		return err
	}
	logger.Info("Finished seeding tax regions.")

	logger.Info("Seeding stock location data...")
	stockLocations, err := stocklocationworkflows.CreateStockLocationsWorkflow.Run(ctx, c, []*dal.Record{rec(
		"name", "European Warehouse",
		"address", rec("city", "Copenhagen", "country_code", "DK", "address_1", ""),
	)})
	if err != nil {
		return err
	}
	stockLocationID := id(stockLocations[0])

	if _, err := link.Create(ctx, []modulessdk.LinkDefinition{modulessdk.NewLinkDefinition(
		utilsmodulessdk.STOCK_LOCATION, "stock_location_id", stockLocationID,
		utilsmodulessdk.FULFILLMENT, "fulfillment_provider_id", "manual_manual",
	)}); err != nil {
		return err
	}

	logger.Info("Seeding fulfillment data...")
	// This is created by a migration script in core.
	shippingProfiles, err := q.Graph(ctx, query.GraphInput{Entity: "shipping_profile", Fields: []string{"id"}}, joiner.Options{})
	if err != nil {
		return err
	}
	if len(shippingProfiles.Data) == 0 {
		return fmt.Errorf("no shipping profile found")
	}
	shippingProfileID := id(shippingProfiles.Data[0].(*dal.Record))

	geoZones := make([]any, len(countries))
	for i, country := range countries {
		geoZones[i] = rec("country_code", country, "type", "country")
	}
	fulfillmentSets, err := fulfillmentModuleService.CreateFulfillmentSets(ctx, []*dal.Record{rec(
		"name", "European Warehouse delivery",
		"type", "shipping",
		"service_zones", []any{rec("name", "Europe", "geo_zones", geoZones)},
	)})
	if err != nil {
		return err
	}
	fulfillmentSet := fulfillmentSets[0]
	serviceZones, _ := fulfillmentSet.Get("service_zones")
	serviceZoneID := id(records(serviceZones)[0])

	if _, err := link.Create(ctx, []modulessdk.LinkDefinition{modulessdk.NewLinkDefinition(
		utilsmodulessdk.STOCK_LOCATION, "stock_location_id", stockLocationID,
		utilsmodulessdk.FULFILLMENT, "fulfillment_set_id", id(fulfillmentSet),
	)}); err != nil {
		return err
	}

	shippingOption := func(name, label, description, code string) *dal.Record {
		return rec(
			"name", name,
			"price_type", "flat",
			"provider_id", "manual_manual",
			"service_zone_id", serviceZoneID,
			"shipping_profile_id", shippingProfileID,
			"type", rec("label", label, "description", description, "code", code),
			"prices", []any{
				rec("currency_code", "usd", "amount", float64(10)),
				rec("currency_code", "eur", "amount", float64(10)),
				rec("region_id", regionID, "amount", float64(10)),
			},
			"rules", []any{
				rec("attribute", "enabled_in_store", "value", "true", "operator", "eq"),
				rec("attribute", "is_return", "value", "false", "operator", "eq"),
			},
		)
	}
	if _, err := fulfillmentworkflows.CreateShippingOptionsWorkflow.Run(ctx, c, []*dal.Record{
		shippingOption("Standard Shipping", "Standard", "Ship in 2-3 days.", "standard"),
		shippingOption("Express Shipping", "Express", "Ship in 24 hours.", "express"),
	}); err != nil {
		return err
	}
	logger.Info("Finished seeding fulfillment data.")

	if _, err := stocklocationworkflows.LinkSalesChannelsToStockLocationWorkflow.Run(ctx, c, stocklocationworkflows.LinkSalesChannelsInput{
		ID:  stockLocationID,
		Add: []string{defaultSalesChannelID},
	}); err != nil {
		return err
	}
	logger.Info("Finished seeding stock location data.")

	logger.Info("Seeding product data...")
	categories, err := productcategoryworkflows.CreateProductCategoriesWorkflow.Run(ctx, c, productcategoryworkflows.CreateProductCategoriesInput{
		ProductCategories: []*dal.Record{
			rec("name", "Shirts", "is_active", true),
			rec("name", "Sweatshirts", "is_active", true),
			rec("name", "Pants", "is_active", true),
			rec("name", "Merch", "is_active", true),
		},
	})
	if err != nil {
		return err
	}
	categoryID := func(name string) string {
		for _, category := range categories {
			if n, _ := category.Get("name"); n == name {
				return id(category)
			}
		}
		return ""
	}

	options, err := productworkflows.CreateProductOptionsWorkflow.Run(ctx, c, productworkflows.CreateProductOptionsInput{
		ProductOptions: []*dal.Record{
			rec("title", "Size", "values", []any{"S", "M", "L", "XL"}),
			rec("title", "Color", "values", []any{"Black", "White"}),
		},
	})
	if err != nil {
		return err
	}
	optionID := func(title string) string {
		for _, option := range options {
			if t, _ := option.Get("title"); t == title {
				return id(option)
			}
		}
		return ""
	}
	sizeOption := rec("id", optionID("Size"))
	colorOption := rec("id", optionID("Color"))

	product := func(title, category, description, handle string, images []any, productOptions []any, variants []any) *dal.Record {
		return rec(
			"title", title,
			"category_ids", []any{categoryID(category)},
			"description", description,
			"handle", handle,
			"weight", float64(400),
			"status", "published",
			"shipping_profile_id", shippingProfileID,
			"images", images,
			"options", productOptions,
			"variants", variants,
			"sales_channels", []any{rec("id", defaultSalesChannelID)},
		)
	}
	if _, err := productworkflows.CreateProductsWorkflow.Run(ctx, c, productworkflows.CreateProductsInput{
		Products: []*dal.Record{
			product("Medusa T-Shirt", "Shirts",
				"Reimagine the feeling of a classic T-shirt. With our cotton T-shirts, everyday essentials no longer have to be ordinary.",
				"t-shirt",
				[]any{imageURL("tee-black-front.png"), imageURL("tee-black-back.png"), imageURL("tee-white-front.png"), imageURL("tee-white-back.png")},
				[]any{sizeOption, colorOption},
				shirtVariants()),
			product("Medusa Sweatshirt", "Sweatshirts",
				"Reimagine the feeling of a classic sweatshirt. With our cotton sweatshirt, everyday essentials no longer have to be ordinary.",
				"sweatshirt",
				[]any{imageURL("sweatshirt-vintage-front.png"), imageURL("sweatshirt-vintage-back.png")},
				[]any{sizeOption},
				sizeVariants("SWEATSHIRT")),
			product("Medusa Sweatpants", "Pants",
				"Reimagine the feeling of classic sweatpants. With our cotton sweatpants, everyday essentials no longer have to be ordinary.",
				"sweatpants",
				[]any{imageURL("sweatpants-gray-front.png"), imageURL("sweatpants-gray-back.png")},
				[]any{sizeOption},
				sizeVariants("SWEATPANTS")),
			product("Medusa Shorts", "Merch",
				"Reimagine the feeling of classic shorts. With our cotton shorts, everyday essentials no longer have to be ordinary.",
				"shorts",
				[]any{imageURL("shorts-vintage-front.png"), imageURL("shorts-vintage-back.png")},
				[]any{sizeOption},
				sizeVariants("SHORTS")),
		},
	}); err != nil {
		return err
	}
	logger.Info("Finished seeding product data.")

	logger.Info("Seeding inventory levels.")
	inventoryItems, err := q.Graph(ctx, query.GraphInput{Entity: "inventory_item", Fields: []string{"id"}}, joiner.Options{})
	if err != nil {
		return err
	}
	levels := make([]*dal.Record, len(inventoryItems.Data))
	for i, item := range inventoryItems.Data {
		levels[i] = rec(
			"location_id", stockLocationID,
			"stocked_quantity", float64(1000000),
			"inventory_item_id", id(item.(*dal.Record)),
		)
	}
	if _, err := inventoryworkflows.CreateInventoryLevelsWorkflow.Run(ctx, c, levels); err != nil {
		return err
	}
	logger.Info("Finished seeding inventory levels data.")
	return nil
}
