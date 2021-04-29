package main

import (
	"net/url"
	"testing"

	"gopkg.in/check.v1"
)

func Test(t *testing.T) { check.TestingT(t) }

type BalancerSuite struct{}

var _ = check.Suite(&BalancerSuite{})

func (s *BalancerSuite) TestBalancer0(c *check.C) {
	serversPool := []Server{
		{Name: "server1:8080", IsAlive: true},
		{Name: "server2:8080", IsAlive: true},
		{Name: "server3:8080", IsAlive: true},
	}

	index0, err := chooseServer(serversPool, &url.URL{Path: "some-path0"})
	c.Check(err, check.IsNil)
	c.Check(index0, check.NotNil)

	index00, err := chooseServer(serversPool, &url.URL{Path: "some-path0"})
	c.Check(err, check.IsNil)
	c.Check(index0, check.NotNil)

	index1, err := chooseServer(serversPool, &url.URL{Path: "some-path1"})
	c.Check(err, check.IsNil)
	c.Check(index1, check.NotNil)
	index2, err := chooseServer(serversPool, &url.URL{Path: "some-path2"})
	c.Check(err, check.IsNil)
	c.Check(index2, check.NotNil)

	c.Check(*index0, check.Equals, *index00)
	c.Check(*index0 == *index1, check.Equals, false)
	c.Check(*index0 == *index2, check.Equals, false)
	c.Check(*index1 == *index2, check.Equals, false)
}

func (s *BalancerSuite) TestBalancer1(c *check.C) {
	serversPool := []Server{
		{Name: "server1:8080", IsAlive: true},
		{Name: "server2:8080", IsAlive: true},
		{Name: "server3:8080", IsAlive: true},
		{Name: "server4:8080", IsAlive: false},
		{Name: "server5:8080", IsAlive: false},
	}

	index0, err := chooseServer(serversPool, &url.URL{Path: "path0"})
	c.Check(err, check.IsNil)
	c.Check(index0, check.NotNil)
	index1, err := chooseServer(serversPool, &url.URL{Path: "path1"})
	c.Check(err, check.IsNil)
	c.Check(index1, check.NotNil)
	index2, err := chooseServer(serversPool, &url.URL{Path: "path2"})
	c.Check(err, check.IsNil)
	c.Check(index2, check.NotNil)

	c.Check(*index0 == 3, check.Equals, false)
	c.Check(*index0 == 4, check.Equals, false)
}

func (s *BalancerSuite) TestBalancer2(c *check.C) {
	serversPool := []Server{
		{Name: "server1:8080", IsAlive: false},
		{Name: "server2:8080", IsAlive: false},
		{Name: "server3:8080", IsAlive: false},
		{Name: "server4:8080", IsAlive: false},
		{Name: "server5:8080", IsAlive: false},
	}

	index0, err := chooseServer(serversPool, &url.URL{Path: "path0"})
	c.Assert(err, check.ErrorMatches, "balancer: no alive servers found")
	c.Check(index0, check.IsNil)
}
