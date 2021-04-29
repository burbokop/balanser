package integration

import (
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"gopkg.in/check.v1"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

func Test(t *testing.T) { check.TestingT(t) }

type IntegrationSuite struct{}

var _ = check.Suite(&IntegrationSuite{})

func sendRequest(id int, c *check.C) string {
	resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data/%d", baseAddress, id))
	c.Check(err, check.IsNil)
	return resp.Header.Get("Lb-From")
}

func (s *IntegrationSuite) TestBalancer0(c *check.C) {
	cnt := map[string]int{}
	for i := 0; i < 1000; i++ {
		lbFrom := sendRequest(int(rand.Uint32()), c)
		c, ok := cnt[lbFrom]
		if ok {
			cnt[lbFrom] = c + 1
		} else {
			cnt[lbFrom] = 0
		}
	}

	for _, i := range cnt {
		c.Check(1000/3-150 < i && i < 1000/3+150, check.Equals, true)
	}
}

func (s *IntegrationSuite) TestBalancer1(c *check.C) {
	id := int(rand.Uint32())
	var first string = sendRequest(id, c)
	for i := 0; i < 999; i++ {
		c.Check(sendRequest(id, c), check.Equals, first)
	}
}

func (s *IntegrationSuite) BenchmarkBalancer(c *check.C) {
	for i := 0; i < c.N; i++ {
		_ = sendRequest(int(rand.Uint32()), c)
	}
}
