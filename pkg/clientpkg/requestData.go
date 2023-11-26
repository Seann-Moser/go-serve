package clientpkg

import (
	"encoding/json"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/pagination"
	"github.com/tidwall/gjson"
	"io"
	"net/http"
)

type RequestData struct {
	Path    string
	Method  string
	Body    interface{}
	Params  map[string]string
	Headers map[string]string
}

func NewRequestData(path, method string, body interface{}, params, headers map[string]string) RequestData {
	return RequestData{
		Path:    path,
		Method:  method,
		Body:    body,
		Params:  params,
		Headers: headers,
	}
}

type ResponseData struct {
	Status int
	Page   *pagination.Pagination
	Err    error
	Data   []byte
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
	if !(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound) {
		rd.Err = fmt.Errorf("invalid Status code: %d", resp.StatusCode)
		return rd
	}
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		rd.Err = err
		return rd
	}
	rd.Page = &pagination.Pagination{}
	err = json.Unmarshal([]byte(gjson.GetBytes(responseData, "page").Raw), &rd.Page)
	if err != nil {
		rd.Err = err
		return rd
	}
	rd.Data = []byte(gjson.GetBytes(responseData, "data").Raw)
	return rd
}
