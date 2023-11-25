package clientpkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/pagination"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"io"

	"net/http"
	"net/url"
)

type Client struct {
	endpoint    *url.URL
	client      *http.Client
	serviceName string
}

func Flags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(prefix, pflag.ExitOnError)
	fs.String(prefix+"-endpoint", "http://127.0.0.1:8080", "")
	fs.String(prefix+"-service-name", "default", "")
	return fs
}

func NewWithFlags(prefix string, client *http.Client) (*Client, error) {
	return New(
		viper.GetString(prefix+"-endpoint"),
		viper.GetString(prefix+"-service-name"),
		client,
	)
}

func New(endpoint, serviceName string, client *http.Client) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	return &Client{
		endpoint:    u,
		client:      client,
		serviceName: serviceName,
	}, nil
}

func (c *Client) SendRequestToEndpoint(ctx context.Context, endpoint *endpoints.Endpoint, method string, body interface{}, params map[string]string, headers map[string]string) ([]byte, *pagination.Pagination, error) {
	return c.SendRequest(ctx, endpoint.URLPath, method, body, params, headers)
}

func GetResponse[T any](body []byte, page *pagination.Pagination, err error) (T, error) {
	var d T
	if err != nil {
		return d, err
	}
	err = json.Unmarshal(body, &d)
	if err != nil {
		return d, err
	}
	return d, nil
}

func (c *Client) SendRequest(ctx context.Context, path string, method string, body interface{}, params map[string]string, headers map[string]string) ([]byte, *pagination.Pagination, error) {
	u, err := url.JoinPath(c.endpoint.String(), path)
	if err != nil {
		return nil, nil, err
	}
	var rawBody []byte
	if body != nil {
		rawBody, err = json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(rawBody))
	if err != nil {
		return nil, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	queryParams := url.Values{}
	for k, v := range params {
		queryParams.Add(k, v)
	}
	req.URL.RawQuery = queryParams.Encode()
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("invalid Status code: %d", resp.StatusCode)
	}
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	var page pagination.Pagination
	_ = json.Unmarshal([]byte(gjson.GetBytes(responseData, "page").Raw), &page)
	return []byte(gjson.GetBytes(responseData, "data").Raw), &page, nil
}
