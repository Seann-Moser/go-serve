package clientpkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.opencensus.io/plugin/ochttp"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/Seann-Moser/go-serve/pkg/pagination"
)

type Client struct {
	endpoint     *url.URL
	client       *http.Client
	serviceName  string
	BackOff      *BackOff
	itemsPerPage uint
	CookieJar    http.CookieJar
	UseCookieJar bool
}

func Flags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(prefix, pflag.ExitOnError)
	fs.String(GetFlagWithPrefix(prefix, "endpoint"), "http://127.0.0.1:8080", fmt.Sprintf("[%s]", strings.ToUpper(ToSnakeCase(GetFlagWithPrefix(prefix, "endpoint")))))
	fs.String(GetFlagWithPrefix(prefix, "service-name"), "default", fmt.Sprintf("[%s]", strings.ToUpper(ToSnakeCase(GetFlagWithPrefix(prefix, "service-name")))))
	fs.Bool(GetFlagWithPrefix(prefix, "use-cookie-jar"), false, fmt.Sprintf("[%s]", strings.ToUpper(ToSnakeCase(GetFlagWithPrefix(prefix, "use-cookie-jar")))))
	fs.Uint(GetFlagWithPrefix(prefix, "items-per-page"), 100, fmt.Sprintf("[%s]", strings.ToUpper(ToSnakeCase(GetFlagWithPrefix(prefix, "items-per-page")))))
	fs.AddFlagSet(BackOffFlags(prefix))
	return fs
}

func NewWithFlags(prefix string, client *http.Client) (*Client, error) {
	return New(
		viper.GetString(GetFlagWithPrefix(prefix, "endpoint")),
		viper.GetString(GetFlagWithPrefix(prefix, "service-name")),
		viper.GetUint(GetFlagWithPrefix(prefix, "items-per-page")),
		viper.GetBool(GetFlagWithPrefix(prefix, "use-cookie-jar")),
		client,
		NewBackoffFromFlags(prefix),
	)
}

func New(endpoint, serviceName string, itemsPerPage uint, useCookieJar bool, client *http.Client, backoff *BackOff) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if itemsPerPage == 0 || itemsPerPage > 1000 {
		itemsPerPage = 100
	}
	if client == nil {
		client = &http.Client{Transport: &ochttp.Transport{
			Base: http.DefaultTransport,
		}}
	}
	if client.Jar == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
		client.Jar = jar
	}
	return &Client{
		endpoint:     u,
		client:       client,
		serviceName:  serviceName,
		BackOff:      backoff,
		itemsPerPage: itemsPerPage,
		CookieJar:    client.Jar,
		UseCookieJar: useCookieJar,
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
	return
}

func (c *Client) SendRequest(ctx context.Context, data RequestData, p *pagination.Pagination) *ResponseData {
	u, err := url.JoinPath(c.endpoint.String(), data.Path)
	if err != nil {
		return &ResponseData{Err: err}
	}

	if p == nil {
		p = &pagination.Pagination{ItemsPerPage: c.itemsPerPage}
	} else {
		p.ItemsPerPage = c.itemsPerPage
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
		req.Header.Set(snakeCaseToHeader(ToSnakeCase(k)), v)
	}

	queryParams := url.Values{}
	data.Params["items_per_page"] = strconv.Itoa(int(p.ItemsPerPage))
	data.Params["page"] = strconv.Itoa(int(p.CurrentPage))

	for k, v := range data.Params {
		queryParams.Add(k, v)
	}
	req.URL.RawQuery = queryParams.Encode()
	resp := NewResponseData(c.client.Do(req))
	if len(resp.Cookies) > 0 && c.UseCookieJar {
		c.CookieJar.SetCookies(c.endpoint, resp.Cookies)
	}
	return resp
}
