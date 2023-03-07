package elasticache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
	"github.com/hashicorp/terraform-provider-aws/names"
)

const (
	ResNameReservedCacheNode = "Reserved Cache Node"
)

// @SDKResource("aws_elasticache_reserved_cache_node")
func ResourceReservedCacheNode() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceReservedCacheNodeCreate,
		ReadWithoutTimeout:   resourceReservedCacheNodeRead,
		UpdateWithoutTimeout: resourceReservedCacheNodeUpdate,
		DeleteWithoutTimeout: resourceReservedCacheNodeDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cache_node_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"duration": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"fixed_price": {
				Type:     schema.TypeFloat,
				Computed: true,
			},
			"cache_node_count": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  1,
			},
			"offering_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"offering_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"product_description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"recurring_charges": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"recurring_charge_amount": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"recurring_charge_frequency": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"reservation_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"start_time": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"usage_price": {
				Type:     schema.TypeFloat,
				Computed: true,
			},
			"tags":     tftags.TagsSchema(),
			"tags_all": tftags.TagsSchemaComputed(),
		},

		CustomizeDiff: verify.SetTagsDiff,
	}
}

func resourceReservedCacheNodeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).ElastiCacheConn()
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(ctx, d.Get("tags").(map[string]interface{})))

	input := &elasticache.PurchaseReservedCacheNodesOfferingInput{
		ReservedCacheNodesOfferingId: aws.String(d.Get("offering_id").(string)),
	}

	if v, ok := d.Get("cache_node_count").(int); ok && v > 0 {
		input.CacheNodeCount = aws.Int64(int64(d.Get("cache_node_count").(int)))
	}

	if v, ok := d.Get("reservation_id").(string); ok && v != "" {
		input.ReservedCacheNodeId = aws.String(v)
	}

	if len(tags) > 0 {
		input.Tags = Tags(tags.IgnoreAWS())
	}

	resp, err := conn.PurchaseReservedCacheNodesOfferingWithContext(ctx, input)
	if err != nil {
		return create.DiagError(names.ElastiCache, create.ErrActionCreating, ResNameReservedCacheNode, fmt.Sprintf("offering_id: %s, reservation_id: %s", d.Get("offering_id").(string), d.Get("reservation_id").(string)), err)
	}

	d.SetId(aws.ToString(resp.ReservedCacheNode.ReservedCacheNodeId))

	if err := waitReservedCacheNodeCreated(ctx, conn, d.Id(), d.Timeout(schema.TimeoutCreate)); err != nil {
		return create.DiagError(names.ElastiCache, create.ErrActionWaitingForCreation, ResNameReservedCacheNode, d.Id(), err)
	}

	return resourceReservedCacheNodeRead(ctx, d, meta)
}

func resourceReservedCacheNodeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).ElastiCacheConn()
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	reservation, err := FindReservedCacheNodeByID(ctx, conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		create.LogNotFoundRemoveState(names.ElastiCache, create.ErrActionReading, ResNameReservedCacheNode, d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return create.DiagError(names.ElastiCache, create.ErrActionReading, ResNameReservedCacheNode, d.Id(), err)
	}

	d.Set("arn", reservation.ReservationARN)
	d.Set("cache_node_type", reservation.CacheNodeType)
	d.Set("duration", reservation.Duration)
	d.Set("fixed_price", reservation.FixedPrice)
	d.Set("cache_node_count", reservation.CacheNodeCount)
	d.Set("offering_id", reservation.ReservedCacheNodesOfferingId)
	d.Set("offering_type", reservation.OfferingType)
	d.Set("product_description", reservation.ProductDescription)
	d.Set("recurring_charges", flattenRecurringCharges(reservation.RecurringCharges))
	d.Set("reservation_id", reservation.ReservedCacheNodeId)
	d.Set("start_time", (reservation.StartTime).Format(time.RFC3339))
	d.Set("state", reservation.State)
	d.Set("usage_price", reservation.UsagePrice)

	tags, err := ListTags(ctx, conn, aws.ToString(reservation.ReservationARN))
	tags = tags.IgnoreAWS().IgnoreConfig(ignoreTagsConfig)

	if err != nil {
		return create.DiagError(names.CE, create.ErrActionReading, ResNameTags, d.Id(), err)
	}

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return create.DiagError(names.CE, create.ErrActionUpdating, ResNameTags, d.Id(), err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return create.DiagError(names.CE, create.ErrActionUpdating, ResNameTags, d.Id(), err)
	}

	return nil
}

func resourceReservedCacheNodeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).ElastiCacheConn()

	if d.HasChange("tags") {
		o, n := d.GetChange("tags")

		if err := UpdateTags(ctx, conn, d.Get("arn").(string), o, n); err != nil {
			return create.DiagError(names.ElastiCache, create.ErrActionUpdating, ResNameTags, d.Id(), err)
		}
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := UpdateTags(ctx, conn, d.Get("arn").(string), o, n); err != nil {
			return create.DiagError(names.ElastiCache, create.ErrActionUpdating, ResNameTags, d.Id(), err)
		}
	}

	return resourceReservedCacheNodeRead(ctx, d, meta)
}

func resourceReservedCacheNodeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Reservations cannot be deleted. Removing from state.
	log.Printf("[DEBUG] %s %s cannot be deleted. Removing from state.: %s", names.ElastiCache, ResNameReservedCacheNode, d.Id())

	return nil
}

func flattenRecurringCharges(recurringCharges []*elasticache.RecurringCharge) []interface{} {
	if len(recurringCharges) == 0 {
		return []interface{}{}
	}

	var rawRecurringCharges []interface{}
	for _, recurringCharge := range recurringCharges {
		rawRecurringCharge := map[string]interface{}{
			"recurring_charge_amount":    recurringCharge.RecurringChargeAmount,
			"recurring_charge_frequency": aws.ToString(recurringCharge.RecurringChargeFrequency),
		}

		rawRecurringCharges = append(rawRecurringCharges, rawRecurringCharge)
	}

	return rawRecurringCharges
}
