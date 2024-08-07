package clientpkg

import (
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/pagination"
	json "github.com/goccy/go-json"
	"github.com/tidwall/gjson"
	"io"
	"net/http"
)

type RequestData struct {
	Path      string
	Method    string
	Body      interface{}
	Params    map[string]string
	Headers   map[string]string
	SkipCache bool
}

func NewRequestData(path, method string, body interface{}, params, headers map[string]string, SkipCache bool) RequestData {
	return RequestData{
		Path:      path,
		Method:    method,
		Body:      body,
		Params:    params,
		Headers:   headers,
		SkipCache: SkipCache,
	}
}

type ResponseData struct {
	Status  int
	Page    *pagination.Pagination
	Message string
	Err     error `json:"-"`
	ErrStr  string
	Data    []byte
	Cookies []*http.Cookie `json:"-"`
}

func NewResponseData(resp *http.Response, err error) *ResponseData {
	if err != nil {
		return &ResponseData{Err: err}
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	rd := &ResponseData{
		Status: resp.StatusCode,
		Page:   nil,
		Err:    nil,
		Data:   nil,
	}
	var responseData []byte
	if resp.Body != nil {
		responseData, err = io.ReadAll(resp.Body)
		if err != nil {
			rd.Err = err
			return rd
		}
		rd.Message = gjson.GetBytes(responseData, "message").Raw
	}
	if !(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound) {
		rd.Err = fmt.Errorf("invalid Status code: %d", resp.StatusCode)
		rd.ErrStr = rd.Err.Error()
		return rd
	}

	rd.Page = &pagination.Pagination{}
	if data := gjson.GetBytes(responseData, "page").Raw; len(data) > 0 {
		err = json.Unmarshal([]byte(data), &rd.Page)
		if err != nil {
			rd.Err = err
			rd.ErrStr = rd.Err.Error()
			return rd
		}
	}
	resp.Cookies()
	rd.Cookies = resp.Cookies()
	rd.Data = []byte(gjson.GetBytes(responseData, "data").Raw)
	return rd
}
