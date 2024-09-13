package request

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type TestStruct struct {
	Code  string `json:"code" xml:"code"`
	State string `json:"state" xml:"state"`
}

func TestGetBody_JSON(t *testing.T) {
	body := `{"code": "123", "state": "active"}`
	req, err := http.NewRequest("POST", "/", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	result, err := GetBody[TestStruct](req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Code != "123" || result.State != "active" {
		t.Fatalf("expected code 123 and state active, got code %s and state %s", result.Code, result.State)
	}
}

func TestGetBody_XML(t *testing.T) {
	body := `<TestStruct><code>123</code><state>active</state></TestStruct>`
	req, err := http.NewRequest("POST", "/", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/xml")

	result, err := GetBody[TestStruct](req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Code != "123" || result.State != "active" {
		t.Fatalf("expected code 123 and state active, got code %s and state %s", result.Code, result.State)
	}
}

func TestGetBody_FormEncoded(t *testing.T) {
	form := url.Values{}
	form.Set("code", "123")
	form.Set("state", "active")

	req, err := http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := GetBody[TestStruct](req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Code != "123" || result.State != "active" {
		t.Fatalf("expected code 123 and state active, got code %s and state %s", result.Code, result.State)
	}
}

func TestGetBody_MultipartForm(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	err := writer.WriteField("code", "123")
	if err != nil {
		t.Fatal(err)
	}
	err = writer.WriteField("state", "active")
	if err != nil {
		t.Fatal(err)
	}
	writer.Close()

	req, err := http.NewRequest("POST", "/", &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	result, err := GetBody[TestStruct](req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Code != "123" || result.State != "active" {
		t.Fatalf("expected code 123 and state active, got code %s and state %s", result.Code, result.State)
	}
}

func TestGetBody_QueryParams(t *testing.T) {
	req, err := http.NewRequest("GET", "/?code=123&state=active", nil)
	if err != nil {
		t.Fatal(err)
	}

	result, err := GetBody[TestStruct](req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Code != "123" || result.State != "active" {
		t.Fatalf("expected code 123 and state active, got code %s and state %s", result.Code, result.State)
	}
}
