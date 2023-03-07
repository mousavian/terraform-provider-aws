package elasticache_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/service/elasticache"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfelasticache "github.com/hashicorp/terraform-provider-aws/internal/service/elasticache"
)

func TestAccElastiCacheReservedCacheNode_basic(t *testing.T) {
	ctx := acctest.Context(t)
	key := "RUN_ELASTICACHE_RESERVED_CACHE_NODE_TESTS"
	vifId := os.Getenv(key)
	if vifId != "true" {
		t.Skipf("Environment variable %s is not set to true", key)
	}

	var reservation elasticache.ReservedCacheNode
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_elasticache_reserved_cache_node.test"
	dataSourceName := "data.aws_elasticache_reserved_cache_node_offering.test"
	cacheNodeCount := "1"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             nil,
		ErrorCheck:               acctest.ErrorCheck(t, elasticache.EndpointsID),
		Steps: []resource.TestStep{
			{
				Config: testAccReservedInstanceConfig_basic(rName, cacheNodeCount),
				Check: resource.ComposeTestCheckFunc(
					testAccReservedInstanceExists(ctx, resourceName, &reservation),
					acctest.MatchResourceAttrRegionalARN(resourceName, "arn", "elasticache", regexp.MustCompile(`reserved-instance:.+`)),
					resource.TestCheckResourceAttrPair(dataSourceName, "cache_node_type", resourceName, "cache_node_type"),
					resource.TestCheckResourceAttrPair(dataSourceName, "duration", resourceName, "duration"),
					resource.TestCheckResourceAttrPair(dataSourceName, "fixed_price", resourceName, "fixed_price"),
					resource.TestCheckResourceAttr(resourceName, "cache_node_count", cacheNodeCount),
					resource.TestCheckResourceAttrPair(dataSourceName, "offering_id", resourceName, "offering_id"),
					resource.TestCheckResourceAttrPair(dataSourceName, "offering_type", resourceName, "offering_type"),
					resource.TestCheckResourceAttrPair(dataSourceName, "product_description", resourceName, "product_description"),
					resource.TestCheckResourceAttrSet(resourceName, "recurring_charges"),
					resource.TestCheckResourceAttr(resourceName, "reservation_id", rName),
					resource.TestCheckResourceAttrSet(resourceName, "start_time"),
					resource.TestCheckResourceAttrSet(resourceName, "state"),
					resource.TestCheckResourceAttrSet(resourceName, "usage_price"),
				),
			},
		},
	})
}

func testAccReservedInstanceExists(ctx context.Context, n string, reservation *elasticache.ReservedCacheNode) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).ElastiCacheConn()

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ElastiCache Reserved Cache Node reservation id is set")
		}

		resp, err := tfelasticache.FindReservedCacheNodeByID(ctx, conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		if resp == nil {
			return fmt.Errorf("ElastiCache Reserved Cache Node %q does not exist", rs.Primary.ID)
		}

		*reservation = *resp

		return nil
	}
}

func testAccReservedInstanceConfig_basic(rName string, cacheNodeCount string) string {
	return fmt.Sprintf(`
data "aws_elasticache_reserved_cache_node_offering" "test" {
  cache_node_type     = "cache.t2.micro"
  duration            = 31536000
  offering_type       = "No Upfront"
  product_description = "redis"
}

resource "aws_elasticache_reserved_cache_node" "test" {
  offering_id      = data.aws_elasticache_reserved_cache_node_offering.test.offering_id
  reservation_id   = %[1]q
  cache_node_count = %[2]s
}
`, rName, cacheNodeCount)
}
