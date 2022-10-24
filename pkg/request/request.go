package request

import (
	"fmt"
	"io/ioutil"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Request struct {
	logger        *zap.Logger
	maxUploadSize int64
}

func NewRequest(maxUploadSize int64, logger *zap.Logger) *Request {
	return &Request{
		logger:        logger,
		maxUploadSize: maxUploadSize,
	}
}

func (req *Request) GetUploadedFile(uploadDir string, r *http.Request) (string, int64, error) {
	if err := r.ParseMultipartForm(req.maxUploadSize); err != nil {
		req.logger.Error("could not parse multipart form", zap.Error(err))
		return "", 0, err
	}
	file, fileHeader, err := r.FormFile("uploadFile")
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = file.Close() }()

	fileSize := fileHeader.Size
	req.logger.Debug(fmt.Sprintf("file size (bytes): %v\n", fileSize))

	if fileSize > req.maxUploadSize {
		return "", fileSize, fmt.Errorf("file was too large %s, max size: %s",
			formatBytes(fileSize),
			formatBytes(req.maxUploadSize))
	}
	fileBytes, err := ioutil.ReadAll(file)
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
	req.logger.Debug(fmt.Sprintf("File_Type: %s, File: %s\n", detectedFileType, newPath))
	if info, err := os.Stat(newPath); err == nil && !info.IsDir() {
		return "", 0, nil
	}
	dir, _ := filepath.Split(newPath)
	err = os.MkdirAll(dir, 0755)
	if err != nil && err != os.ErrExist {
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
