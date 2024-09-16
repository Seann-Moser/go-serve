package generator

import (
	_ "embed"
	"fmt"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var _ Generator = GoClientGenerator{}

type GoClientGenerator struct {
}

var defaultImports = []Imports{
	{
		Name: "",
		Path: "context",
	},
	{
		Name: "",
		Path: "net/http",
	},
	{
		Name: "clientpkg",
		Path: "github.com/Seann-Moser/go-serve/pkg/clientpkg",
	},
	{
		Name: "",
		Path: "fmt",
	},
}

/*
Todo update gofunc template to support better godoc comments
*/
const goFuncTemplate = `
// {{.Name}}
// {{.Description}} {{.Swagger}}
func (c *Client) {{.Name}}(ctx context.Context{{if .RequestType}}, {{.RequestTypeName}} {{.RequestType}}{{end}}{{range .MuxVars}}, {{.}} string{{end}}{{if .UsesQueryParams }}{{range $k, $v := .QueryParams}}, {{$v}} string{{end}}{{end}}{{if .UsesHeaderParams }}, headers map[string]string{{end}},skipCache bool) {{.Return}} {
	path := {{.Path}}{{if .UsesQueryParams }}
	params := map[string]string{}{{end}}{{range $k, $v := .QueryParams}}
	params["{{$k}}"] = {{$v}}{{end}}
	requestDataInternal := clientpkg.NewRequestData(path, http.Method{{.MethodType}}, {{if .RequestType}}{{.RequestTypeName}}{{else}}nil{{end}}, {{if .UsesQueryParams }} params{{else}}nil{{end}}, {{if .UsesHeaderParams }} clientpkg.MergeMap[string](headers,c.headers){{else}}clientpkg.MergeMap[string](nil,c.headers){{end}},skipCache){{if .UseIterator}}
	return clientpkg.NewIterator[{{.DataTypeName}}](ctx, c.base, requestDataInternal){{ else }}
	return c.base.Request(ctx,requestDataInternal,nil,true){{ end }}
}
`

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
}

func (cf *ClientFunc) GenerateSwaggerDoc() string {
	var sb strings.Builder

	// General endpoint summary and description
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("// @Summary %s\n", cf.Name))
	sb.WriteString(fmt.Sprintf("// @Description %s\n", cf.Description))
	sb.WriteString(fmt.Sprintf("// @Tags %s\n", cf.Name))
	sb.WriteString(fmt.Sprintf("// @Accept json\n"))
	sb.WriteString(fmt.Sprintf("// @Produce json\n"))

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

	// Generate Swagger for header params
	if cf.UsesHeaderParams {
		// You can add specific header params as needed
		sb.WriteString("// @Param Authorization header string true \"Bearer Token\"\n")
	}

	// Request body documentation if RequestType exists
	if cf.RequestType != "" {
		sb.WriteString(fmt.Sprintf("// @Param data body %s true \"%s request body\"\n", cf.RequestTypeName, cf.Name))
	}

	// Return type (success response)
	sb.WriteString(fmt.Sprintf("// @Success 200 {object} %s\n", cf.DataTypeName))

	// If an iterator is used, mark the response as a stream (or handle as needed)
	if cf.UseIterator {
		sb.WriteString("// @Router /endpoint [get] // Use appropriate method type here\n")
	}

	// HTTP method type (GET, POST, etc.)
	sb.WriteString(fmt.Sprintf("// @Router %s [%s]", cf.RawPath, strings.ToLower(cf.MethodType)))
	cf.Swagger = sb.String()
	return cf.Swagger
}

//go:embed templates/struct_template.tmpl
var clientTemplates string

func (g GoClientGenerator) Generate(data GeneratorData, endpoint ...*endpoints.Endpoint) error {
	groupedEndpoints := groupEndpointsByGroup(endpoint) // Group by group name
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	homeDir = path.Join(homeDir, "go", "src", "") + "/"
	publicDir := filepath.Join(homeDir, data.RootDir, "pkg", ToSnakeCase(data.ProjectName+"-client"))
	privateDir := filepath.Join(homeDir, data.RootDir, "pkg", ToSnakeCase(fmt.Sprintf("%s-client_private", data.ProjectName)))
	for group, endpoints := range groupedEndpoints {
		// Create public and private directories

		// Create directories if they don't exist
		if err := ensureDir(publicDir); err != nil {
			return err
		}
		if err := ensureDir(privateDir); err != nil {
			return err
		}

		var functions []string
		var PublicFunctions []string
		var imports []Imports
		var publicImports []Imports
		// Generate and write each endpoint function for both public and private dirs
		for _, ep := range endpoints {

			for _, v := range GoNewClientFunc(ep) {
				funcCode, err := generateEndpointFunc(v)
				if err != nil {
					return err
				}
				imports = append(imports, v.Imports...)
				functions = append(functions, funcCode)
				if ep.Public {
					PublicFunctions = append(PublicFunctions, funcCode)
					publicImports = append(publicImports, v.Imports...)
				}
			}

		}
		// Write function to files

		if err := writeToFile(publicDir, group, PublicFunctions, true, publicImports...); err != nil {
			return err
		}

		if err := writeToFile(privateDir, group, functions, false, imports...); err != nil {
			return err
		}

	}
	/*
		"context"
		"fmt"
		"github.com/Seann-Moser/go-serve/pkg/clientpkg"
		"github.com/Seann-Moser/go-serve/pkg/response"
		"github.com/spf13/pflag"
		"net/http"
		"time"
	*/
	clientImports := []Imports{
		{
			Path: "context",
		},
		{
			Path: "fmt",
		},
		{
			Path: "github.com/Seann-Moser/go-serve/pkg/clientpkg",
		},
		{
			Path: "github.com/Seann-Moser/go-serve/pkg/response",
		},
		{
			Path: "github.com/spf13/pflag",
		},
		{
			Path: "net/http",
		},
		{
			Path: "time",
		},
	}

	clientTemplates, err := templ(map[string]interface{}{
		"Name":    data.ProjectName,
		"Headers": []string{},
	}, clientTemplates)
	if err := writeToFile(publicDir, "client", []string{clientTemplates}, true, clientImports...); err != nil {
		return err
	}
	if err := writeToFile(privateDir, "client", []string{clientTemplates}, true, clientImports...); err != nil {
		return err
	}

	return nil
}
