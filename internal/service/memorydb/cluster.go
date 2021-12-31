package memorydb

import (
	"context"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/memorydb"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceClusterCreate,
		ReadContext:   resourceClusterRead,
		UpdateContext: resourceClusterUpdate,
		DeleteContext: resourceClusterDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(clusterAvailableTimeout),
			Update: schema.DefaultTimeout(clusterAvailableTimeout),
			Delete: schema.DefaultTimeout(clusterDeletedTimeout),
		},

		CustomizeDiff: verify.SetTagsDiff,

		Schema: map[string]*schema.Schema{
			"acl_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"auto_minor_version_upgrade": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},
			"cluster_endpoint": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"port": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
			"engine_patch_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"engine_version": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"kms_key_arn": {
				// The API will accept an ID, but return the ARN on every read.
				// For the sake of consistency, force everyone to use ARN-s.
				// To prevent confusion, the attribute is suffixed _arn rather
				// than the _id implied by the API.
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidARN,
			},
			"maintenance_window": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: verify.ValidOnceAWeekWindowFormat,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 40),
					validation.StringDoesNotMatch(
						regexp.MustCompile(`[-][-]`),
						"The name may not contain two consecutive hyphens."),
					validation.StringMatch(
						// Similar to ElastiCache, MemoryDB normalises names to lowercase.
						regexp.MustCompile(`^[a-z0-9-]*[a-z0-9]$`),
						"Only lowercase alphanumeric characters and hyphens allowed. The name may not end with a hyphen."),
				),
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 40-resource.UniqueIDSuffixLength),
					validation.StringDoesNotMatch(
						regexp.MustCompile(`[-][-]`),
						"The name may not contain two consecutive hyphens."),
					validation.StringMatch(
						// Similar to ElastiCache, MemoryDB normalises names to lowercase.
						regexp.MustCompile(`^[a-z0-9-]+$`),
						"Only lowercase alphanumeric characters and hyphens allowed."),
				),
			},
			"node_type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"number_of_shards": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"parameter_group_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"security_group_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"snapshot_retention_limit": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"snapshot_window": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"sns_topic_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"subnet_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"tags":     tftags.TagsSchema(),
			"tags_all": tftags.TagsSchemaComputed(),
			"tls_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},
		},
	}
}

func resourceClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).MemoryDBConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(d.Get("tags").(map[string]interface{})))

	name := create.Name(d.Get("name").(string), d.Get("name_prefix").(string))
	input := &memorydb.CreateClusterInput{
		ACLName:                 aws.String(d.Get("acl_name").(string)),
		AutoMinorVersionUpgrade: aws.Bool(d.Get("auto_minor_version_upgrade").(bool)),
		ClusterName:             aws.String(name),
		NodeType:                aws.String(d.Get("node_type").(string)),
		Tags:                    Tags(tags.IgnoreAWS()),
		TLSEnabled:              aws.Bool(d.Get("tls_enabled").(bool)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("engine_version"); ok {
		input.EngineVersion = aws.String(v.(string))
	}

	if v, ok := d.GetOk("kms_key_arn"); ok {
		input.KmsKeyId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("maintenance_window"); ok {
		input.MaintenanceWindow = aws.String(v.(string))
	}

	if v, ok := d.GetOk("subnet_group_name"); ok {
		input.SubnetGroupName = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating MemoryDB Cluster: %s", input)
	_, err := conn.CreateClusterWithContext(ctx, input)

	if err != nil {
		return diag.Errorf("error creating MemoryDB Cluster (%s): %s", name, err)
	}

	if err := waitClusterAvailable(ctx, conn, name); err != nil {
		return diag.Errorf("error waiting for MemoryDB Cluster (%s) to be created: %s", name, err)
	}

	d.SetId(name)

	return resourceClusterRead(ctx, d, meta)
}

func resourceClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).MemoryDBConn

	if d.HasChangesExcept("tags", "tags_all") {
		input := &memorydb.UpdateClusterInput{
			ClusterName: aws.String(d.Id()),
		}

		if d.HasChange("acl_name") {
			input.ACLName = aws.String(d.Get("acl_name").(string))
		}

		if d.HasChange("description") {
			input.Description = aws.String(d.Get("description").(string))
		}

		if d.HasChange("engine_version") {
			input.EngineVersion = aws.String(d.Get("engine_version").(string))
		}

		if d.HasChange("maintenance_window") {
			input.MaintenanceWindow = aws.String(d.Get("maintenance_window").(string))
		}

		if d.HasChange("node_type") {
			input.NodeType = aws.String(d.Get("node_type").(string))
		}

		log.Printf("[DEBUG] Updating MemoryDB Cluster (%s)", d.Id())

		_, err := conn.UpdateClusterWithContext(ctx, input)
		if err != nil {
			return diag.Errorf("error updating MemoryDB Cluster (%s): %s", d.Id(), err)
		}

		if err := waitClusterAvailable(ctx, conn, d.Id()); err != nil {
			return diag.Errorf("error waiting for MemoryDB Cluster (%s) to be modified: %s", d.Id(), err)
		}
	}

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := UpdateTags(conn, d.Get("arn").(string), o, n); err != nil {
			return diag.Errorf("error updating MemoryDB Cluster (%s) tags: %s", d.Id(), err)
		}
	}

	return resourceClusterRead(ctx, d, meta)
}

func resourceClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).MemoryDBConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	cluster, err := FindClusterByName(ctx, conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] MemoryDB Cluster (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.Errorf("error reading MemoryDB Cluster (%s): %s", d.Id(), err)
	}

	d.Set("acl_name", cluster.ACLName)
	d.Set("arn", cluster.ARN)
	d.Set("auto_minor_version_upgrade", cluster.AutoMinorVersionUpgrade)

	if v := cluster.ClusterEndpoint; v != nil {
		m := map[string]interface{}{}
		if v := aws.StringValue(v.Address); v != "" {
			m["address"] = v
		}
		if v := aws.Int64Value(v.Port); v != 0 {
			m["port"] = v
		}
		d.Set("cluster_endpoint", []interface{}{m})
	}

	d.Set("description", cluster.Description)
	d.Set("engine_patch_version", cluster.EnginePatchVersion)
	d.Set("engine_version", cluster.EngineVersion)
	d.Set("kms_key_arn", cluster.KmsKeyId) // KmsKeyId is actually an ARN here.
	d.Set("maintenance_window", cluster.MaintenanceWindow)
	d.Set("name", cluster.Name)
	d.Set("name_prefix", create.NamePrefixFromName(aws.StringValue(cluster.Name)))
	d.Set("node_type", cluster.NodeType)
	d.Set("number_of_shards", cluster.NumberOfShards)
	d.Set("parameter_group_name", cluster.ParameterGroupName)

	var securityGroupIds []*string
	for _, v := range cluster.SecurityGroups {
		securityGroupIds = append(securityGroupIds, v.SecurityGroupId)
	}
	d.Set("security_group_ids", flex.FlattenStringSet(securityGroupIds))

	d.Set("snapshot_retention_limit", cluster.SnapshotRetentionLimit)
	d.Set("snapshot_window", cluster.SnapshotWindow)
	d.Set("sns_topic_arn", cluster.SnsTopicArn)
	d.Set("subnet_group_name", cluster.SubnetGroupName)
	d.Set("tls_enabled", cluster.TLSEnabled)

	tags, err := ListTags(conn, d.Get("arn").(string))

	if err != nil {
		return diag.Errorf("error listing tags for MemoryDB Cluster (%s): %s", d.Id(), err)
	}

	tags = tags.IgnoreAWS().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return diag.Errorf("error setting tags for MemoryDB Cluster (%s): %s", d.Id(), err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return diag.Errorf("error setting tags_all for MemoryDB Cluster (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).MemoryDBConn

	log.Printf("[DEBUG] Deleting MemoryDB Cluster: (%s)", d.Id())
	_, err := conn.DeleteClusterWithContext(ctx, &memorydb.DeleteClusterInput{
		ClusterName: aws.String(d.Id()),
	})

	if tfawserr.ErrCodeEquals(err, memorydb.ErrCodeClusterNotFoundFault) {
		return nil
	}

	if err != nil {
		return diag.Errorf("error deleting MemoryDB Cluster (%s): %s", d.Id(), err)
	}

	if err := waitClusterDeleted(ctx, conn, d.Id()); err != nil {
		return diag.Errorf("error waiting for MemoryDB Cluster (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}
