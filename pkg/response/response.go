package response

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Seann-Moser/go-serve/pkg/pagination"
	"go.uber.org/zap"
)

type Response struct {
	logger    *zap.Logger
	showError bool
}
type BaseResponse struct {
	Message string                 `json:"message"`
	Data    interface{}            `json:"data,omitempty"`
	Page    *pagination.Pagination `json:"page,omitempty"`
}

func NewResponse(showErr bool, logger *zap.Logger) *Response {
	return &Response{logger: logger, showError: showErr}
}

func (resp *Response) Error(w http.ResponseWriter, err error, code int, message string) {
	w.WriteHeader(code)

	resp.logger.Error(message, zap.Error(err), zap.Int("code", code))
	var dataErr error
	if err != nil && resp.showError {
		dataErr = err
	}
	EncodeErr := json.NewEncoder(w).Encode(BaseResponse{
		Message: message,
		Data:    dataErr,
	})
	if EncodeErr != nil {
		resp.logger.Error("failed encoding response", zap.Error(EncodeErr))
	}
}

func (resp *Response) PaginationResponse(w http.ResponseWriter, data interface{}, page *pagination.Pagination) {
	d, err := json.Marshal(data)
	if err != nil {
		resp.logger.Error("failed to marshall data", zap.Error(err))
		return
	}
	resp.logger.Info(string(d))
	var pageData []interface{}
	err = json.Unmarshal(d, &pageData)
	if err != nil {
		resp.logger.Error("failed to encode to []interface", zap.Error(err))
		return
	}
	w.WriteHeader(http.StatusOK)
	bytes, err := json.MarshalIndent(BaseResponse{
		Data: getRange(pageData, page),
		Page: page,
	}, "", "    ")
	if err != nil {
		resp.logger.Error("failed to encode response")
	}
	_, EncodeErr := w.Write(bytes)
	if EncodeErr != nil {
		resp.logger.Error("failed encoding response", zap.Error(EncodeErr))
	}
}
func getRange(data []interface{}, page *pagination.Pagination) []interface{} {
	page.TotalItems = uint(len(data))
	if page.ItemsPerPage == 0 {
		page.ItemsPerPage = pagination.MaxItemsPerPage
	}
	if page.TotalItems < page.ItemsPerPage {
		page.TotalPages = 1
	} else {
		page.TotalPages = uint(math.Ceil(float64(page.TotalItems) / float64(page.ItemsPerPage)))
	}
	page.NextPage = page.CurrentPage + 1
	if page.NextPage > page.TotalPages {
		page.NextPage = page.TotalPages
	}
	if page.CurrentPage > page.TotalPages {
		page.CurrentPage = page.TotalPages
	}
	if len(data) < int(page.ItemsPerPage) {
		return data
	}
	min := int((page.CurrentPage - 1) * page.ItemsPerPage)
	max := min + int(page.ItemsPerPage)
	fmt.Printf("Min %d Max %d\n", min, max)
	if min > len(data) {
		return []interface{}{}
	}
	if max > len(data) {
		return data[min:]
	}
	return data[min:max]
}

func (resp *Response) Message(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusOK)
	bytes, err := json.MarshalIndent(BaseResponse{
		Message: msg,
	}, "", "    ")
	if err != nil {
		resp.logger.Error("failed to encode response")
	}
	_, EncodeErr := w.Write(bytes)
	if EncodeErr != nil {
		resp.logger.Error("failed encoding response", zap.Error(EncodeErr))
	}
}
func (resp *Response) Raw(w http.ResponseWriter, r *http.Response) {
	w.WriteHeader(r.StatusCode)
	if r.Body != nil {
		defer r.Body.Close()
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			resp.logger.Error("failed reading body", zap.Error(err))
			return
		}
		for k, v := range r.Header {
			w.Header().Set(k, v[0])
		}
		_, err = w.Write(b)
		if err != nil {
			resp.logger.Error("failed encoding response", zap.Error(err))
			return
		}
	}
}
func (resp *Response) DataResponse(w http.ResponseWriter, data interface{}, code int) {
	w.WriteHeader(code)
	bytes, err := json.MarshalIndent(BaseResponse{
		Data: data,
	}, "", "    ")
	if err != nil {
		resp.logger.Error("failed to encode response")
	}
	_, EncodeErr := w.Write(bytes)
	if EncodeErr != nil {
		resp.logger.Error("failed encoding response", zap.Error(EncodeErr))
	}
}

func (resp *Response) File(w http.ResponseWriter, file string, download bool) (int64, error) {
	if info, err := os.Stat(file); err != nil || info.IsDir() {
		return 0, err
	}
	filename := strings.Split(file, "/")
	w.Header().Set("filename", filename[len(filename)-1])
	if download {
		w.Header().Set("Content-Description", "File Transfer")
		w.Header().Set("Content-Transfer-Encoding", "binary")
		w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(filename[len(filename)-1]))
		//w.Header().Set("Content-Type", "application/octet-stream")
	}
	f, _ := os.Open(file)
	defer func() {
		_ = f.Close()
	}()

	fileHeader := make([]byte, 512)
	_, err := f.Read(fileHeader)
	if err != nil {
		return 0, err
	}
	fileStat, _ := f.Stat()
	w.Header().Set("Content-Type", http.DetectContentType(fileHeader))
	w.Header().Set("Content-Length", strconv.FormatInt(fileStat.Size(), 10))
	_, err = f.Seek(0, 0)
	if err != nil {
		return 0, err
	}
	return io.Copy(w, f)
}