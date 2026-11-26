package migrationscripts

import (
	"context"
	"fmt"
	"log/slog"

	regionworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/region/workflows"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	typescommon "github.com/hitkey-io/medusago/packages/core/types/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"

	"backend/src/lib"
)

// RegionStorePolicies adds the store policy defaults that a region's
// metadata misses, keeping the values a merchant already set.
func RegionStorePolicies(ctx context.Context, args typescommon.ExecArgs) error {
	c := args.Container
	logger := c.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
	q := c.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)

	regions, err := q.Graph(ctx, query.GraphInput{Entity: "region", Fields: []string{"id", "name", "metadata"}}, joiner.Options{})
	if err != nil {
		return err
	}
	defaults := lib.DefaultRegionStorePolicies()
	for _, row := range regions.Data {
		region := row.(*dal.Record)
		v, _ := region.Get("metadata")
		metadata, _ := v.(*dal.Record)
		merged := dal.NewRecord()
		if metadata != nil {
			for _, k := range metadata.Keys() {
				value, _ := metadata.Get(k)
				merged.Set(k, value)
			}
		}
		missing := 0
		for _, k := range defaults.Keys() {
			if _, ok := merged.Get(k); ok {
				continue
			}
			value, _ := defaults.Get(k)
			merged.Set(k, value)
			missing++
		}
		if missing == 0 {
			continue
		}
		if _, err := regionworkflows.UpdateRegionsWorkflow.Run(ctx, c, regionworkflows.UpdateRegionsInput{
			Selector: rec("id", id(region)),
			Update:   rec("metadata", merged),
		}); err != nil {
			return err
		}
		name, _ := region.Get("name")
		logger.Info(fmt.Sprintf("Added store policy defaults to region \"%v\"", name))
	}
	return nil
}
