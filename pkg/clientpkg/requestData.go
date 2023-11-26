package clientpkg

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
