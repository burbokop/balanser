package integration

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/burbokop/balanser/httptools"
	"gopkg.in/check.v1"
)

type DBValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func getValue(key string, c *check.C) DBValue {
	responce, err := client.Get(fmt.Sprintf("%s/api/v1/some-data/0?key=%s", baseAddress, key))
	c.Check(err, check.IsNil)
	result := &DBValue{}
	err = httptools.DecodeBodyAndClose(responce.Body, result)
	c.Check(err, check.IsNil)
	return *result
}

func setValue(value DBValue, c *check.C) {
	type Body struct {
		Value string `json:"value"`
	}
	data, err := json.Marshal(Body{Value: value.Value})
	c.Check(err, check.IsNil)
	_, err = client.Post(
		fmt.Sprintf("%s/api/v1/some-data?key=%s", baseAddress, value.Key),
		"application/json",
		bytes.NewReader(data),
	)
	c.Check(err, check.IsNil)
}

func (s *IntegrationSuite) TestDB1(c *check.C) {
	setValue(DBValue{Key: "team-name", Value: "gogadoda"}, c)
	result := getValue("team-name", c)

	c.Check(result.Value, check.Equals, "gogadoda")
}
