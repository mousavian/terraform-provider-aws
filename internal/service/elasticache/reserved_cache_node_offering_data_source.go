package elasticache

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/names"
)

const (
	ResNameReservedCacheNodeOffering = "Reserved Cache Node Offering"
)

// @SDKDataSource("aws_elasticache_reserved_cache_node_offering")
func DataSourceReservedCacheNodeOffering() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataSourceReservedCacheNodeOfferingRead,
		Schema: map[string]*schema.Schema{
			"cache_node_type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"duration": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"fixed_price": {
				Type:     schema.TypeFloat,
				Computed: true,
			},
			"offering_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"offering_type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"Light Utilization",
					"Medium Utilization",
					"Heavy Utilization",
					"Partial Upfront",
					"All Upfront",
					"No Upfront",
				}, false),
			},
			"product_description": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceReservedCacheNodeOfferingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).ElastiCacheConn()

	input := &elasticache.DescribeReservedCacheNodesOfferingsInput{
		CacheNodeType:      aws.String(d.Get("cache_node_type").(string)),
		Duration:           aws.String(fmt.Sprint(d.Get("duration").(int))),
		OfferingType:       aws.String(d.Get("offering_type").(string)),
		ProductDescription: aws.String(d.Get("product_description").(string)),
	}

	resp, err := conn.DescribeReservedCacheNodesOfferingsWithContext(ctx, input)

	if err != nil {
		return create.DiagError(names.ElastiCache, create.ErrActionReading, ResNameReservedCacheNodeOffering, "unknown", err)
	}

	if len(resp.ReservedCacheNodesOfferings) == 0 {
		return diag.Errorf("no %s %s found matching criteria; try different search", names.ElastiCache, ResNameReservedCacheNodeOffering)
	}

	if len(resp.ReservedCacheNodesOfferings) > 1 {
		return diag.Errorf("More than one %s %s found matching criteria; try different search", names.ElastiCache, ResNameReservedCacheNodeOffering)
	}

	offering := resp.ReservedCacheNodesOfferings[0]

	d.SetId(aws.ToString(offering.ReservedCacheNodesOfferingId))
	d.Set("cache_node_type", offering.CacheNodeType)
	d.Set("duration", offering.Duration)
	d.Set("fixed_price", offering.FixedPrice)
	d.Set("offering_type", offering.OfferingType)
	d.Set("product_description", offering.ProductDescription)
	d.Set("offering_id", offering.ReservedCacheNodesOfferingId)

	return nil
}
