package generators

import (
	_ "embed"
	"fmt"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	"os"
	"path"
	"path/filepath"
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
var defaultImportsJS = []Imports{
	{
		Name: "assets/",
		Path: "context",
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

//go:embed templates/struct_template.tmpl
var clientTemplates string

func (g GoClientGenerator) Generate(data GeneratorData, endpoint ...*endpoints.Endpoint) error {
	groupedEndpoints := groupEndpointsByGroup(endpoint) // Group by group name
	publicDir, privateDir, err := GetPublicPrivateDir(data)
	if err != nil {
		return err
	}

	for group, endpoints := range groupedEndpoints {
		// Create public and private directories

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

		if err := writeToGoFile(publicDir, group, PublicFunctions, true, publicImports...); err != nil {
			return err
		}

		if err := writeToGoFile(privateDir, group, functions, false, imports...); err != nil {
			return err
		}

	}

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
	if err != nil {
		return err
	}
	if err := writeToGoFile(publicDir, "client", []string{clientTemplates}, true, clientImports...); err != nil {
		return err
	}
	if err := writeToGoFile(privateDir, "client", []string{clientTemplates}, true, clientImports...); err != nil {
		return err
	}

	return nil
}

func GetPublicPrivateDir(data GeneratorData) (string, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}

	homeDir = path.Join(homeDir, "go", "src", "") + "/"
	publicDir := filepath.Join(homeDir, data.RootDir, "pkg", ToSnakeCase(data.ProjectName+"-client"))
	privateDir := filepath.Join(homeDir, data.RootDir, "pkg", ToSnakeCase(fmt.Sprintf("%s-client_private", data.ProjectName)))
	// Create directories if they don't exist
	if err := ensureDir(publicDir); err != nil {
		return "", "", err
	}
	if err := ensureDir(privateDir); err != nil {
		return "", "", err
	}
	return publicDir, privateDir, nil
}
