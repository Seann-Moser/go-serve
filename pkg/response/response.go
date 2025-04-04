package response

import (
	"bytes"
	"encoding/json"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/Seann-Moser/go-serve/pkg/pagination"
)

func NewResponse(showErr bool) *Response {
	return &Response{showError: showErr}
}

var skip = []zap.Option{zap.AddCallerSkip(1)}

func (resp *Response) Error(r *http.Request, w http.ResponseWriter, err error, code int, message string) {
	w.WriteHeader(code)
	if err != nil {
		logger := ctxLogger.GetLogger(r.Context())
		logger.WithOptions(skip...).Error(message, zap.Error(err), zap.Int("code", code))
	}
	var dataErr error
	if err != nil && resp.showError {
		dataErr = err
	}
	EncodeErr := BaseResponse{
		Message: message,
		Data:    dataErr,
	}.Encode(r, w)
	if EncodeErr != nil {
		logger := ctxLogger.GetLogger(r.Context())
		logger.WithOptions(skip...).Warn("failed encoding response", zap.Error(EncodeErr))
	}
}

func (resp *Response) PaginationResponse(r *http.Request, w http.ResponseWriter, data interface{}, page *pagination.Pagination) {
	d, err := json.Marshal(data)
	if err != nil {
		ctxLogger.Error(r.Context(), "failed to marshall data", zap.Error(err))
		return
	}
	var pageData []interface{}
	err = json.Unmarshal(d, &pageData)
	if err != nil {
		ctxLogger.Error(r.Context(), "failed to encode to []interface", zap.Error(err))
		return
	}

	w.WriteHeader(http.StatusOK)
	err = BaseResponse{
		Data: getRange(pageData, page, false),
		Page: page,
	}.Encode(r, w)
	if err != nil {
		ctxLogger.Warn(r.Context(), "failed encoding response", zap.Error(err))
	}
}

func (resp *Response) RawPaginationResponse(r *http.Request, w http.ResponseWriter, data interface{}, page *pagination.Pagination, totalItems uint) {
	d, err := json.Marshal(data)
	if err != nil {
		ctxLogger.Error(r.Context(), "failed to marshall data", zap.Error(err))
		return
	}
	var pageData []interface{}
	err = json.Unmarshal(d, &pageData)
	if err != nil {
		ctxLogger.Error(r.Context(), "failed to encode to []interface", zap.Error(err))
		return
	}
	page.TotalItems = uint(totalItems)
	if page.CurrentPage <= 0 {
		page.CurrentPage = 1
	}

	if page.ItemsPerPage == 0 {
		page.ItemsPerPage = pagination.MaxItemsPerPage
	}
	if page.TotalItems < page.ItemsPerPage {
		page.TotalPages = 1
	} else {
		page.TotalPages = uint(math.Ceil(float64(page.TotalItems) / float64(page.ItemsPerPage)))
	}
	if page.CurrentPage > page.TotalPages {
		page.CurrentPage = page.TotalPages
	}
	w.WriteHeader(http.StatusOK)
	err = BaseResponse{
		Data: getRange(pageData, page, true),
		Page: page,
	}.Encode(r, w)
	if err != nil {
		ctxLogger.Error(r.Context(), "failed to encode response", zap.Error(err))

	}

}

func getRange(data []interface{}, page *pagination.Pagination, raw bool) []interface{} {
	if raw {
		if page.ItemsPerPage == 0 {
			page.ItemsPerPage = pagination.MaxItemsPerPage
		}
		if page.CurrentPage <= 0 {
			page.CurrentPage = 1
		}
		page.NextPage = page.CurrentPage + 1
		if page.NextPage > page.TotalPages {
			page.NextPage = page.TotalPages
		}
		if page.CurrentPage > page.TotalPages {
			page.CurrentPage = page.TotalPages
		}
		if page.TotalItems < page.ItemsPerPage {
			page.TotalPages = 1
		} else {
			page.TotalPages = uint(math.Ceil(float64(page.TotalItems) / float64(page.ItemsPerPage)))
		}
		if int(page.ItemsPerPage) > len(data) && page.CurrentPage >= page.TotalPages {
			page.ItemsPerPage = uint(len(data))
		}
		return data
	}

	page.TotalItems = uint(len(data))
	if page.ItemsPerPage == 0 {
		page.ItemsPerPage = pagination.MaxItemsPerPage
	}
	if page.ItemsPerPage <= 0 {
		page.ItemsPerPage = 1
	}
	if page.CurrentPage <= 0 {
		page.CurrentPage = 1
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
	if min < 0 {
		min = 0
	}
	max := min + int(page.ItemsPerPage)
	if min > len(data) {
		return []interface{}{}
	}
	if max > len(data) {
		return data[min:]
	}
	return data[min:max]
}

func (resp *Response) Message(r *http.Request, w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusOK)

	err := BaseResponse{
		Message: msg,
	}.Encode(r, w)
	if err != nil {
		ctxLogger.Error(r.Context(), "failed to encode response", zap.Error(err))
	}
}

func (resp *Response) Raw(req *http.Request, w http.ResponseWriter, r *http.Response) {
	w.WriteHeader(r.StatusCode)
	if r.Body != nil {
		defer r.Body.Close()
		b, err := io.ReadAll(r.Body)
		if err != nil {
			ctxLogger.Error(req.Context(), "failed reading body", zap.Error(err))
			return
		}
		for k, v := range r.Header {
			w.Header().Set(k, v[0])
		}
		_, err = w.Write(b)
		if err != nil {
			ctxLogger.Error(req.Context(), "failed encoding response", zap.Error(err))
			return
		}
	}
}
func (resp *Response) DataResponse(r *http.Request, w http.ResponseWriter, data interface{}, code int) {
	w.WriteHeader(code)
	err := BaseResponse{
		Data: data,
	}.Encode(r, w)
	if err != nil {
		ctxLogger.Error(r.Context(), "failed to encode response", zap.Error(err))
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

func (resp *Response) ByteFile(w http.ResponseWriter, filename string, file []byte, download bool) (int64, error) {
	// Set headers for the file
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(file)), 10))
	w.Header().Set("Content-Type", http.DetectContentType(file))

	if download {
		w.Header().Set("Content-Description", "File Transfer")
		w.Header().Set("Content-Transfer-Encoding", "binary")
		w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(filename))
		// Optionally use application/octet-stream for downloads
		// w.Header().Set("Content-Type", "application/octet-stream")
	} else {
		w.Header().Set("filename", filename)
	}

	// Write file to response
	w.WriteHeader(http.StatusOK) // Explicitly send HTTP 200 OK status
	return io.Copy(w, bytes.NewBuffer(file))
}

func (resp *Response) DataNoWrap(r *http.Request, w http.ResponseWriter, data interface{}, code int) {
	w.WriteHeader(code)
	bytes, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		ctxLogger.Error(r.Context(), "failed to encode response")
	}
	_, EncodeErr := w.Write(bytes)
	if EncodeErr != nil {
		ctxLogger.Error(r.Context(), "failed encoding response", zap.Error(EncodeErr))
	}
}

//func writeResponse(ctx context.Context, data interface{}, msg string, err error, code int, contentType string, nowrap bool, isArray bool, w http.ResponseWriter) {
//	bytes, err := json.MarshalIndent(BaseResponse{
//		Data: data,
//	}, "", "    ")
//
//	switch contentType {
//	case "application/json":
//		//case "application/x-www-form-urlencoded":
//
//	}
//	if nowrap{
//		data = BaseResponse{
//			Data: data,
//		}
//	}
//	bytes, err := json.MarshalIndent(BaseResponse{
//		Data: data,
//	}, "", "    ")
//	if err != nil {
//		ctxLogger.Error(r.Context(), "failed to encode response")
//	}
//	err = json.NewEncoder(w).Encode(data)
//	if EncodeErr != nil {
//		ctxLogger.Error(ctx, "failed encoding response", zap.Error(err))
//	}
//
//}
