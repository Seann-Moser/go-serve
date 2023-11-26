package clientpkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/pagination"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"io"
	"strconv"

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

func (c *Client) SendRequest(ctx context.Context, data RequestData, p *pagination.Pagination) ([]byte, *pagination.Pagination, error) {
	u, err := url.JoinPath(c.endpoint.String(), data.Path)
	if err != nil {
		return nil, nil, err
	}
	if data.Headers == nil {
		data.Headers = map[string]string{}
	}
	if data.Params == nil {
		data.Params = map[string]string{}
	}
	var rawBody []byte
	if data.Body != nil {
		rawBody, err = json.Marshal(data.Body)
		if err != nil {
			return nil, nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, data.Method, u, bytes.NewReader(rawBody))
	if err != nil {
		return nil, nil, err
	}
	for k, v := range data.Headers {
		req.Header.Set(k, v)
	}
	queryParams := url.Values{}
	data.Params["items_per_page"] = strconv.Itoa(int(p.ItemsPerPage))
	data.Params["page"] = strconv.Itoa(int(p.CurrentPage))

	for k, v := range data.Params {
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
