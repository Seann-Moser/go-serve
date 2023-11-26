package clientpkg

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/Seann-Moser/go-serve/pkg/pagination"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"strconv"

	"net/http"
	"net/url"
)

type Client struct {
	endpoint    *url.URL
	client      *http.Client
	serviceName string
	BackOff     *BackOff
}

func Flags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(prefix, pflag.ExitOnError)
	fs.String(prefix+"-endpoint", "http://127.0.0.1:8080", "")
	fs.String(prefix+"-service-name", "default", "")
	fs.AddFlagSet(BackOffFlags(prefix))
	return fs
}

func NewWithFlags(prefix string, client *http.Client) (*Client, error) {
	return New(
		viper.GetString(prefix+"-endpoint"),
		viper.GetString(prefix+"-service-name"),
		client,
		NewBackoffFromFlags(prefix),
	)
}

func New(endpoint, serviceName string, client *http.Client, backoff *BackOff) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	return &Client{
		endpoint:    u,
		client:      client,
		serviceName: serviceName,
		BackOff:     backoff,
	}, nil
}

func (c *Client) RequestWithRetry(ctx context.Context, data RequestData, p *pagination.Pagination) (resp *ResponseData) {
	_ = c.BackOff.Retry(ctx, func() error {
		resp = c.SendRequest(ctx, data, p)
		if resp.Status == http.StatusTooManyRequests {
			return resp.Err
		}
		return nil
	})

	return nil
}

func (c *Client) SendRequest(ctx context.Context, data RequestData, p *pagination.Pagination) *ResponseData {
	u, err := url.JoinPath(c.endpoint.String(), data.Path)
	if err != nil {
		return &ResponseData{Err: err}
	}
	if p == nil {
		p = &pagination.Pagination{ItemsPerPage: 100}
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
			return &ResponseData{Err: err}
		}
	}

	req, err := http.NewRequestWithContext(ctx, data.Method, u, bytes.NewReader(rawBody))
	if err != nil {
		return &ResponseData{Err: err}
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
	return NewResponseData(c.client.Do(req))
}
