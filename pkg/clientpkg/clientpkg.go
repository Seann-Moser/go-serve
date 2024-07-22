package clientpkg

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Seann-Moser/ctx_cache"
	json "github.com/goccy/go-json"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/Seann-Moser/go-serve/pkg/pagination"
)

var _ HttpClient = &Client{}
var _ HttpClient = &MockClient{}

var UseResponseCache = false

type HttpClient interface {
	Request(ctx context.Context, data RequestData, p *pagination.Pagination, retry bool) (resp *ResponseData)
	RequestWithRetry(ctx context.Context, data RequestData, p *pagination.Pagination) (resp *ResponseData)
	SendRequest(ctx context.Context, data RequestData, p *pagination.Pagination) *ResponseData
}

type MockClient struct {
}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (m MockClient) defaultResponse() *ResponseData {
	return &ResponseData{
		Status: http.StatusOK,
		Page: &pagination.Pagination{
			CurrentPage:  1,
			NextPage:     0,
			TotalItems:   0,
			TotalPages:   0,
			ItemsPerPage: 0,
		},
		Message: "",
		Err:     nil,
		Data:    []byte{},
	}
}

func (m MockClient) Request(ctx context.Context, data RequestData, p *pagination.Pagination, retry bool) *ResponseData {
	return m.defaultResponse()
}

func (m MockClient) RequestWithRetry(ctx context.Context, data RequestData, p *pagination.Pagination) *ResponseData {
	return m.defaultResponse()
}

func (m MockClient) SendRequest(ctx context.Context, data RequestData, p *pagination.Pagination) *ResponseData {
	return m.defaultResponse()
}

type Client struct {
	endpoint     *url.URL
	client       *http.Client
	serviceName  string
	BackOff      *BackOff
	itemsPerPage uint
	CookieJar    http.CookieJar
	UseCookieJar bool
	skipCache    bool
}

func Flags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(prefix, pflag.ExitOnError)
	fs.String(GetFlagWithPrefix(prefix, "endpoint"), "http://127.0.0.1:8080", fmt.Sprintf("[%s]", strings.ToUpper(ToSnakeCase(GetFlagWithPrefix(prefix, "endpoint")))))
	fs.String(GetFlagWithPrefix(prefix, "service-name"), "default", fmt.Sprintf("[%s]", strings.ToUpper(ToSnakeCase(GetFlagWithPrefix(prefix, "service-name")))))
	fs.Bool(GetFlagWithPrefix(prefix, "use-cookie-jar"), false, fmt.Sprintf("[%s]", strings.ToUpper(ToSnakeCase(GetFlagWithPrefix(prefix, "use-cookie-jar")))))
	fs.Bool(GetFlagWithPrefix(prefix, "skip-cache"), false, fmt.Sprintf("[%s]", strings.ToUpper(ToSnakeCase(GetFlagWithPrefix(prefix, "use-cookie-jar")))))
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
		client = &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
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
func (c *Client) SkipCache(skip bool) {
	c.skipCache = skip
}
func (c *Client) CacheKey(data RequestData, p *pagination.Pagination) string {
	var key string
	key = fmt.Sprintf("%s%s%s%s%s", key, c.endpoint, data.Path, data.Method, MapToString(data.Params))
	if p != nil {
		key = fmt.Sprintf("%s%d%d", key, p.CurrentPage, p.ItemsPerPage)
	}
	return key
}

func (c *Client) Request(ctx context.Context, data RequestData, p *pagination.Pagination, retry bool) (resp *ResponseData) {
	if retry {
		return c.RequestWithRetry(ctx, data, p)
	}
	key := c.CacheKey(data, p)
	if strings.EqualFold(data.Method, http.MethodGet) && !c.skipCache && !strings.Contains(data.Path, "healthcheck") && !data.SkipCache {
		resp, err := ctx_cache.GetSetP[ResponseData](ctx, 15*time.Second, c.serviceName, key, func(ctx context.Context) (*ResponseData, error) {
			resp = c.SendRequest(ctx, data, p)
			if resp.Status == http.StatusTooManyRequests {
				return nil, resp.Err
			}
			return resp, nil
		})
		if resp != nil && err == nil {
			return resp
		}
	}
	return c.SendRequest(ctx, data, p)
}

func (c *Client) RequestWithRetry(ctx context.Context, data RequestData, p *pagination.Pagination) (resp *ResponseData) {
	key := c.CacheKey(data, p)

	if strings.EqualFold(data.Method, http.MethodGet) && !c.skipCache && !strings.Contains(data.Path, "healthcheck") && !data.SkipCache {
		resp, err := ctx_cache.GetSetP[ResponseData](ctx, 15*time.Second, c.serviceName, key, func(ctx context.Context) (*ResponseData, error) {
			resp = c.SendRequest(ctx, data, p)
			if resp.Status == http.StatusTooManyRequests {
				return nil, resp.Err
			}
			return resp, nil
		})
		if resp != nil && err == nil {
			return resp
		}
	}
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
		return &ResponseData{Err: err, ErrStr: err.Error()}
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

func MapToString(m map[string]string) string {
	var output []string
	for k, v := range m {
		output = append(output, fmt.Sprintf("%s-%s", k, v))
	}
	return strings.Join(output, "-")
}
