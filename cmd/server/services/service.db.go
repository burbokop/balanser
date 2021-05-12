package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/burbokop/balanser/httptools"
)

type DBValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Done string

var done Done = "done"

type DBService interface {
	GetValue(key string) (*DBValue, error)
	SetValue(key string, value string) (*Done, error)
}

type DefaultDBService struct {
	baseAddress string
	client      http.Client
}

func NewDefaultDBService(baseAddress string) *DefaultDBService {
	return &DefaultDBService{
		baseAddress: baseAddress,
		client: http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (service *DefaultDBService) GetValue(key string) (*DBValue, error) {
	responce, err := service.client.Get(fmt.Sprintf("%s/db/%s", service.baseAddress, key))
	if err != nil {
		return nil, fmt.Errorf("error from db server: %s", err)
	}

	result := &DBValue{}
	err = httptools.DecodeBodyAndClose(responce.Body, result)
	if err != nil {
		return nil, fmt.Errorf("error decoding responce: %s", err)
	}
	return result, nil
}

func (service *DefaultDBService) SetValue(key string, value string) (*Done, error) {
	type Body struct {
		Value string `json:"value"`
	}
	data, err := json.Marshal(Body{Value: value})
	if err != nil {
		return nil, fmt.Errorf("error encoding json: %s", err)
	}

	_, err = service.client.Post(
		fmt.Sprintf("%s/db/%s", service.baseAddress, key),
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		if err != nil {
			return nil, fmt.Errorf("error from db server: %s", err)
		}
	}
	return &done, nil
}
