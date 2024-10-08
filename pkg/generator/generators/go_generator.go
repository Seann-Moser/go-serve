package generators

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	openai "github.com/sashabaranov/go-openai"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var _ Generator = GoClientGenerator{}

type GoClientGenerator struct {
	client  *openai.Client
	headers []string
}

func NewGoClientGenerator(client *openai.Client) *GoClientGenerator {
	return &GoClientGenerator{client: client}
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
}
var defaultImportsJS = []Imports{
	{
		Name: "assets/",
		Path: "context",
	},
}

var SwagApiGeneralTempl string = `
	@title {{.Title}}
	@version {{.Version}}
	@description {{.Description}}
	
	@contact.name API Support
	@contact.url https://support.surveynoodle.com
	@contact.email support@surveynoodle.com
	
	@schemes http https
	@host {{.Host}}
	@BasePath /
	@query.collection.format multi
	
	@externalDocs.description  OpenAPI
	@externalDocs.url          https://support.surveynoodle.com
	@securitydefinitions.oauth2.application OAuth2Application
	@tokenUrl https://iam.surveynoodle.com/oauth/token
	@authorizationurl https://iam.surveynoodle.com/oauth/authorize
`

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

func (g GoClientGenerator) AddHeader(key, value string) {
	if g.headers == nil {
		g.headers = make([]string, 0)
	}
	g.headers = append(g.headers, key)
}
func (g GoClientGenerator) Generate(data GeneratorData, endpoint ...*endpoints.Endpoint) error {
	groupedEndpoints := groupEndpointsByGroup(endpoint) // Group by group name
	publicDir, privateDir, err := GetPublicPrivateDir(data)
	if err != nil {
		return err
	}

	for group, endpoints := range groupedEndpoints {
		g.GenerateComments(data, endpoints...)
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
		"Headers": g.headers,
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

func (g GoClientGenerator) GenerateComments(data GeneratorData, epts ...*endpoints.Endpoint) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	cachedEndpoints := ".cache.json"

	homeDir = path.Join(homeDir, "go", "src", "") + "/"
	cachedEndpoints = filepath.Join(homeDir, data.RootDir, cachedEndpoints)
	foundEndpoints, err := LoadEndpoints(cachedEndpoints)
	if foundEndpoints == nil {
		foundEndpoints = map[string]*endpoints.Endpoint{}
	}
	goFiles := GetGoFiles(filepath.Join(homeDir, data.RootDir))

	api, err := templ(map[string]string{
		"Title":       data.Title,
		"Version":     data.Version,
		"Description": data.Description,
		"Host":        data.Host,
	}, SwagApiGeneralTempl)
	if err != nil {
		return
	}

	d := FindFunction("main", goFiles)
	for _, v := range d {
		v.Comment.Set(api)
		_ = v.UpdateComment()
		break
	}
	for _, e := range epts {
		if e == nil {
			continue
		}
		fullName := GetFunctionName(e.HandlerFunc)
		if fullName == "" {
			continue
		}
		if e.Description == "" {
			e.Description = ""
		}
		fullName = strings.TrimSuffix(fullName, "-fm")
		_, pkg := path.Split(fullName)
		tmpPkg := strings.Split(pkg, ".")
		pkg = strings.Join(tmpPkg[1:], " ")
		pkg = strings.TrimPrefix(pkg, "(*")
		pkg = strings.ReplaceAll(pkg, ")", `\)`)
		tmp := FindFunction(pkg, goFiles)
		d := sortByLongestFunction(tmp)
		if len(d) == 0 {
			continue
		}
		hash := StringToBase64(d[0].Func.Data)
		recalc := false
		if end, found := foundEndpoints[e.UniqueID()]; found {
			recalc = end.Hash != hash || end.Hash == "" || end.Description == ""
			if !recalc {
				e.Hash = end.Hash
				e.Description = end.Description
			}
		} else {
			recalc = true
		}

		if recalc {
			e.Hash = hash
			_, _ = generateDescriptions(g.client, e, tmp)

		}

	}

	_ = SaveEndpoints(cachedEndpoints, epts)
}

// StringToBase64 removes all formatting (spaces, newlines, tabs) and converts a plain string to a Base64-encoded string.
func StringToBase64(input string) string {
	// Remove all newlines, tabs, spaces, and any other unnecessary formatting
	re := regexp.MustCompile(`\s+`) // Matches any whitespace (spaces, newlines, tabs)
	cleanedInput := re.ReplaceAllString(input, "")

	// Convert the cleaned input string to a byte slice
	data := []byte(cleanedInput)

	// Encode the byte slice to Base64
	base64Str := base64.StdEncoding.EncodeToString(data)

	return base64Str
}

// SaveEndpoints saves an array of endpoints to a file in JSON format.
func SaveEndpoints(file string, endpoints []*endpoints.Endpoint) error {
	// Marshal the array of endpoints to JSON
	data, err := json.MarshalIndent(endpoints, "", "  ")
	if err != nil {
		return err
	}

	// Write the JSON data to the file
	err = os.WriteFile(file, data, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func LoadEndpoints(file string) (map[string]*endpoints.Endpoint, error) {
	// Check if file exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, errors.New("file does not exist")
	}

	// Read the file content
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	// Unmarshal the file content to []*endpoints.Endpoint
	var endpointList []*endpoints.Endpoint
	if err := json.Unmarshal(data, &endpointList); err != nil {
		return nil, err
	}
	output := make(map[string]*endpoints.Endpoint)
	for _, ep := range endpointList {
		output[ep.UniqueID()] = ep
	}
	return output, nil
}

type funcEntry struct {
	Key  string
	Func Func
}

// Define a function to sort by length of comment or name
func sortByLongestFunction(funcs map[string]Func) []funcEntry {
	// Convert map to slice
	var entries []funcEntry
	for key, fn := range funcs {
		entries = append(entries, funcEntry{Key: key, Func: fn})
	}

	// Sort the slice by length of the comment or function name
	sort.Slice(entries, func(i, j int) bool {
		// You can choose to sort by function name length or comment length
		// Here we use length of the comment's lines as an example
		lenI := len(entries[i].Func.Comment.Lines)
		lenJ := len(entries[j].Func.Comment.Lines)
		return lenI < lenJ // Sort in descending order
	})

	return entries
}

func generateDescriptions(client *openai.Client, endpoints *endpoints.Endpoint, funcs map[string]Func) (map[string]string, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	d := sortByLongestFunction(funcs)
	if len(d) == 0 {
		return nil, nil
	}
	descriptions := make(map[string]string)
	fn := d[0].Func
	// Generate prompt based on the Func struct
	prompt := fmt.Sprintf(
		"Describe the following function in 1-2 sentences, this should give a summary of the function:\n\nFile: %s\nName: %s\nLine: %d\nFunction: '''%v''''",
		fn.File, fn.Name, fn.Ln, fn.Data,
	)
	req := openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: prompt,
			},
		},
		MaxTokens: 150,
	}
	// Call OpenAI's API
	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("completion error:%w", err)
	}

	// Store the description in the map
	descriptions[d[0].Key] = resp.Choices[0].Message.Content
	endpoints.Description = resp.Choices[0].Message.Content

	return descriptions, nil
}
