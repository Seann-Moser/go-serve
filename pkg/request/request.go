package request

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"io"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Request struct {
	maxUploadSize int64
}

func NewRequest(maxUploadSize int64) *Request {
	return &Request{
		maxUploadSize: maxUploadSize,
	}
}

func (req *Request) DownloadFile(headerName string, uploadDir string, r *http.Request) (string, int64, error) {
	if err := r.ParseMultipartForm(req.maxUploadSize); err != nil {
		ctxLogger.Error(r.Context(), "could not parse multipart form", zap.Error(err))
		return "", 0, err
	}
	file, fileHeader, err := r.FormFile(headerName)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = file.Close() }()

	fileSize := fileHeader.Size
	ctxLogger.Debug(r.Context(), fmt.Sprintf("file size (bytes): %v\n", fileSize))

	if fileSize > req.maxUploadSize {
		return "", fileSize, fmt.Errorf("file was too large %s, max size: %s",
			formatBytes(fileSize),
			formatBytes(req.maxUploadSize))
	}
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", 0, err
	}
	detectedFileType := http.DetectContentType(fileBytes)
	switch detectedFileType {
	case "image/jpeg", "image/jpg":
	case "image/gif", "image/png":
	case "application/pdf":
		break
	default:
		return "", 0, fmt.Errorf("invalid file type: %s", detectedFileType)
	}
	fileName := uuid.New().String()
	if filename := r.Header.Get("filename"); filename != "" {
		fileName = filename
	}
	fileEndings, err := mime.ExtensionsByType(detectedFileType)
	if err != nil {
		return "", 0, err
	}

	newPath := filepath.Join(uploadDir, fileName+fileEndings[0])
	ctxLogger.Debug(r.Context(), fmt.Sprintf("File_Type: %s, File: %s\n", detectedFileType, newPath))
	if info, err := os.Stat(newPath); err == nil && !info.IsDir() {
		return "", 0, nil
	}
	dir, _ := filepath.Split(newPath)
	err = os.MkdirAll(dir, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return "", 0, err
	}
	newFile, err := os.Create(newPath)
	if err != nil {
		return newPath, 0, err
	}

	if _, err := newFile.Write(fileBytes); err != nil {
		return "", 0, err
	}
	return newPath, int64(len(fileBytes)), nil
}

func (req *Request) GetUploadedFile(uploadDir string, r *http.Request) (string, int64, error) {
	return req.DownloadFile("uploadFile", uploadDir, r)
}

func formatBytes(b int64) string {
	var suffixes [5]string
	suffixes[0] = "B"
	suffixes[1] = "KB"
	suffixes[2] = "MB"
	suffixes[3] = "GB"
	suffixes[4] = "TB"
	base := math.Log(float64(b)) / math.Log(1024)
	getSize := round(math.Pow(1024, base-math.Floor(base)), .5, 2)
	getSuffix := suffixes[int(math.Floor(base))]
	return strconv.FormatFloat(getSize, 'f', -1, 64) + " " + string(getSuffix)
}

func round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

func GetBody[T any](r *http.Request) (*T, error) {
	var d T
	header := r.Header.Get("Content-Type")
	if strings.Contains(header, ";") {
		header = strings.Split(header, ";")[0]
	}
	// Handle different content types
	switch header {
	case "application/json":
		// Decode JSON body
		err := json.NewDecoder(r.Body).Decode(&d)
		if err != nil {
			return nil, fmt.Errorf("failed decoding body(%s): %w", r.Header.Get("Content-Type"), err)
		}
		return &d, nil

	case "application/xml":
		// Decode XML body
		err := xml.NewDecoder(r.Body).Decode(&d)
		if err != nil {
			return nil, fmt.Errorf("failed decoding XML body(%s): %w", r.Header.Get("Content-Type"), err)
		}
		return &d, nil

	case "application/x-www-form-urlencoded":
		// Parse form-encoded body
		err := r.ParseForm()
		if err != nil {
			return nil, fmt.Errorf("failed parsing form-encoded body: %w", err)
		}
		// Use reflection to populate fields from the form data
		if err := populateFromFormValues(&d, r.Form); err != nil {
			return nil, err
		}
		return &d, nil

	case "multipart/form-data":
		// Parse multipart form data
		err := r.ParseMultipartForm(10 << 20) // 10 MB max memory
		if err != nil {
			return nil, fmt.Errorf("failed parsing multipart form data: %w", err)
		}
		// Use reflection to populate fields from the multipart form data
		if err := populateFromFormValues(&d, r.MultipartForm.Value); err != nil {
			return nil, err
		}
		return &d, nil

	default:
		// Fallback to query parameters
		queryParams := r.URL.Query()
		if err := populateFromFormValues(&d, queryParams); err != nil {
			return nil, err
		}
		return &d, nil
	}
}

// populateFromFormValues uses reflection to populate struct fields from form or query parameters
func populateFromFormValues[T any](d *T, values map[string][]string) error {
	val := reflect.ValueOf(d).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Get the field name or the JSON tag name
		name := fieldType.Tag.Get("json")
		if name == "" {
			name = fieldType.Name
		}

		// Check if the form or query parameters contain the field
		if v, ok := values[name]; ok {
			// Set the field value (assuming it's a string for simplicity, extend as needed)
			if field.Kind() == reflect.String {
				field.SetString(v[0])
			}
			// Handle other types as needed (e.g., int, bool, etc.)
		}
	}
	return nil
}
