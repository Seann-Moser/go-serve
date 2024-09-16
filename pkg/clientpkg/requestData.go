package clientpkg

import (
	"encoding/json"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/pagination"
	"github.com/Seann-Moser/go-serve/pkg/request"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"io"
	"net/http"
	"os"
	"path"
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
	Status   int
	Page     *pagination.Pagination
	Message  string
	Err      error `json:"-"`
	ErrStr   string
	Data     []byte
	Cookies  []*http.Cookie `json:"-"`
	FilePath string
}

func (d *ResponseData) Close() {
	if d.FilePath != "" {
		if _, err := os.Stat(d.FilePath); os.IsNotExist(err) {
			return
		}
		_ = os.Remove(d.FilePath)
	}

}

// Helper: Ensure directory exists or create it
func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, os.ModePerm)
	}
	return nil
}
func IsImageFile(resp *http.Response) (bool, string) {
	switch resp.Header.Get("Content-Type") {
	case "image/jpeg":
		return true, ".jpg"
	case "image/png":
		return true, ".png"
	case "image/gif":
		return true, ".gif"
	case "image/webp":
		return true, ".webp"
	case "image/bmp":
		return true, ".bmp"
	case "image/tiff":
		return true, ".tiff"
	case "image/x-icon":
		return true, ".ico"
	case "image/svg+xml":
		return true, ".svg"
	default:
		return false, ""
	}
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

		isImage, ext := IsImageFile(resp)
		if isImage {
			name := uuid.New().String() + ext
			dir := "/" + path.Join("tmp", "img")
			_ = ensureDir(dir)
			p := path.Join(dir, name)

			err = request.DownloadImageFromResponse(resp, p)
			if err != nil {
				rd.Err = err
				return rd
			}
			rd.FilePath = p
		}
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
