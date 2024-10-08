package generators

import (
	"fmt"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	"strings"
)

type Language string

const (
	LanguageGo = Language("go")
	LanguageJS = Language("js")
)

type GeneratorData struct {
	ProjectName string
	RootDir     string
	Title       string
	Version     string
	Description string
	Host        string
}

type Generator interface {
	Generate(data GeneratorData, endpoints ...*endpoints.Endpoint) error
}

type ClientFunc struct {
	Name          string
	UrlEnvVarName string
	Return        string

	RawPath     string
	Path        string
	MethodType  string
	MuxVars     []string
	UseIterator bool

	UsesQueryParams  bool
	UsesHeaderParams bool
	RequestType      string
	Async            bool
	RequestTypeName  string
	DataTypeName     string
	QueryParams      map[string]string
	Description      string
	Imports          []Imports

	Objects map[string][]string
	Swagger string

	Language Language
}

func (cf *ClientFunc) GenerateSwaggerDoc() string {
	var sb strings.Builder
	parts := strings.Split(cf.Path, "/")
	tags := cf.Name
	if len(parts) > 1 {
		tags = parts[1]
	}
	// General endpoint summary and description
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("// @Summary %s\n", cf.Name))
	sb.WriteString(fmt.Sprintf("// @Description %s\n", cf.Description))
	sb.WriteString(fmt.Sprintf("// @Tags %s\n", tags))
	sb.WriteString("// @Accept json\n")
	sb.WriteString("// @Produce json\n")

	// Generate Swagger for path params (MuxVars)
	for _, muxVar := range cf.MuxVars {
		sb.WriteString(fmt.Sprintf("// @Param %s path string true \"%s\"\n", muxVar, muxVar))
	}

	// Generate Swagger for query params
	if cf.UsesQueryParams {
		for param, paramName := range cf.QueryParams {
			sb.WriteString(fmt.Sprintf("// @Param %s query string false \"%s\"\n", paramName, param))
		}
	}
	//@securitydefinitions.oauth2.application OAuth2Application
	// Generate Swagger for header params
	if cf.UsesHeaderParams {
		// You can add specific header params as needed
		sb.WriteString("// @Param Authorization header string true \"Bearer Token\"\n")
	}

	// Request body documentation if RequestType exists
	if cf.RequestType != "" {
		sb.WriteString(fmt.Sprintf("// @Param data body %s true \"%s request body\"\n", strings.ReplaceAll(strings.ReplaceAll(cf.RequestType, "*", ""), " ", ""), cf.Name))
	}

	// Return type (success response)
	if cf.DataTypeName != "" {
		//cf.MethodType
		//if isMap(requestType) {
		//	cf.RequestTypeName = fmt.Sprintf("Map<%s>", typeName)
		//}

		sb.WriteString(fmt.Sprintf("// @Success 200 {object} %s\n", strings.ReplaceAll(strings.ReplaceAll(cf.DataTypeName, "*", ""), " ", "")))
	}

	// If an iterator is used, mark the response as a stream (or handle as needed)
	if cf.UseIterator {
		sb.WriteString("// @Router /endpoint [get] // Use appropriate method type here\n")
	}

	// HTTP method type (GET, POST, etc.)
	sb.WriteString(fmt.Sprintf("// @Router %s [%s]", cf.RawPath, strings.ToLower(cf.MethodType)))
	cf.Swagger = sb.String()
	return cf.Swagger
}
