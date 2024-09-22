package generators

import (
	"bytes"
	"fmt"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"math/rand/v2"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func GetRootDir() (string, error) {
	currentPath, err := os.Getwd()
	if err != nil {
		return "", err
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	homeDir = path.Join(homeDir, "go", "src", "") + "/"
	rootDir := ""
	count := 0
	for _, i := range regexp.MustCompile(`[/\\]`).Split(strings.ReplaceAll(currentPath, homeDir, ""), -1) {
		rootDir = path.Join(rootDir, i)
		if count > 1 {

			break
		}
		count++
	}
	return rootDir, nil
}

func GetProjectName() (string, error) {
	rootDir, err := GetRootDir()
	if err != nil {
		return "", fmt.Errorf("failed to get project name: %v", err)
	}
	_, projectName := path.Split(rootDir)
	return projectName, nil
}

func SnakeCaseToCamelCase(inputUnderScoreStr string) (camelCase string) {
	//snake_case to camelCase

	isToUpper := false

	for k, v := range inputUnderScoreStr {
		if k == 0 {
			camelCase = strings.ToUpper(string(inputUnderScoreStr[0]))
		} else {
			if isToUpper {
				camelCase += strings.ToUpper(string(v))
				isToUpper = false
			} else {
				if v == '_' {
					isToUpper = true
				} else {
					camelCase += string(v)
				}
			}
		}
	}
	return

}

func ToSnakeCase(str string) string {
	str = strings.ReplaceAll(str, "-", "_")
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(strings.ReplaceAll(snake, "__", "_"))
}
func getTypePkg(myVar interface{}) string {
	switch myVar.(type) {
	case string:
		return "string"
	case int64:
		return "int64"
	case []string:
		return "[]string"
	}
	t := reflect.TypeOf(myVar)
	if t == nil {
		return ""
	}
	if isArray(myVar) {
		return t.Elem().PkgPath()
	}
	if t.Kind() == reflect.Ptr {
		return t.Elem().PkgPath()
	}
	return t.PkgPath()
}

func isArray(myVar interface{}) bool {
	if myVar == nil {
		return false
	}
	t := reflect.TypeOf(myVar)
	if t == nil {
		return false
	}
	switch t.Kind() {
	case reflect.Slice:
		return true
	case reflect.Array:
		return true
	case reflect.Ptr:
		return isArray(t.Elem())
	default:
		return false
	}

}
func isMap(i interface{}) bool {
	switch i.(type) {
	case map[string]string, map[string]interface{}, map[interface{}]interface{}, map[int64]interface{}, map[int64]string:
		return true
	}
	return false
}

func getType(myVar interface{}) string {
	switch myVar.(type) {
	case string:
		return "string"
	case int64:
		return "int64"
	case []string:
		return "string"
	}

	t := reflect.TypeOf(myVar)
	if t == nil {
		return "interface{}"
	}
	if t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		if t.Name() == "" {
			return t.Elem().Name()
		}
		return t.Name()
	}
}

func UrlToName(url string) string {
	re := regexp.MustCompile(`\{(.*?)\}`)
	for _, d := range re.FindAllString(url, -1) {
		url = strings.ReplaceAll(url, "/"+d, "")

	}
	url = strings.TrimPrefix(url, "/")
	url = strings.TrimSuffix(url, "/")

	url = strings.ReplaceAll(url, "/", "_")
	return SnakeCaseToCamelCase(url)
}

func formatName(name string, isMap bool) string {
	formattedName := SnakeCaseToCamelCase(ToSnakeCase(name))
	if isMap {
		formattedName += "Map"
	}
	return strings.ToLower(formattedName[:1]) + formattedName[1:]
}

func getRequestTypeString(requestType interface{}, pkg string) string {
	if isMap(requestType) {
		return fmt.Sprintf("map[%s]%s", reflect.TypeOf(requestType).Key(), reflect.TypeOf(requestType).Elem())
	} else if isArray(requestType) {
		return fmt.Sprintf("[]*%s.%s", pkg, getType(requestType))
	} else {
		return fmt.Sprintf("*%s.%s", pkg, getType(requestType))
	}
}

func getDataTypeName(responseType interface{}, pkg string, skipPkg map[string]bool) string {
	if isMap(responseType) {
		return fmt.Sprintf("map[%s]%s", reflect.TypeOf(responseType).Key(), reflect.TypeOf(responseType).Elem())
	} else if _, found := skipPkg[pkg]; found {
		return pkg
	} else if isArray(responseType) {
		return fmt.Sprintf("%s.%s", pkg, getType(responseType))
	} else {
		return fmt.Sprintf("%s.%s", pkg, getType(responseType))
	}
}

func GetBaseDir(path string) string {
	// Split the path by "/"
	parts := strings.Split(path, "/")

	// Loop through the parts and find the first non-empty element
	for _, part := range parts {
		if part != "" {
			return part
		}
	}
	return ""
}

// Helper: Ensure directory exists or create it
func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, os.ModePerm)
	}
	return nil
}

// Helper: Generate function code for an endpoint using the template
func generateEndpointFunc(ep *ClientFunc) (string, error) {
	ep.GenerateSwaggerDoc()
	tmpl, err := template.New("goFuncTemplate").Parse(goFuncTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ep); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func templ(ep interface{}, tmp string) (string, error) {
	tmpl, err := template.New("tmpl").Parse(tmp)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ep); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// writeToGoFile Helper: Write generated function code to a file
func writeToGoFile(dir string, group string, code []string, isPublic bool, imports ...Imports) error {
	if len(code) == 0 {
		return nil
	}
	filename := fmt.Sprintf("%s.go", ToSnakeCase(group))
	fp := filepath.Join(dir, filename)

	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()

	_, pkg := path.Split(dir)
	header := fmt.Sprintf(
		`package %s
// AUTO GENERATED
import (
	%s
)

`, ToSnakeCase(pkg), FormatImports(LanguageGo, imports...))
	// Add package name and imports at the top of the file
	if isPublic {
		header += "// Public Endpoint - Auto Generated\n"
	} else {
		header += "// Private Endpoint - Auto Generated\n"
	}

	_, err = f.WriteString(header + strings.Join(code, "\n") + "\n")
	return err
}

// writeNuxtFile Helper: Write generated function code to a file
func writeNuxtFile(dir string, group string, code []string, isPublic bool, imports ...Imports) error {
	if len(code) == 0 {
		return nil
	}
	filename := fmt.Sprintf("%s_plugin.js", ToSnakeCase(group))
	fp := filepath.Join(dir, filename)

	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()

	_, pkg := path.Split(dir)
	header := fmt.Sprintf(
		` // %s
// AUTO GENERATED
%s


`, ToSnakeCase(pkg), FormatImports(LanguageJS, imports...))
	// Add package name and imports at the top of the file
	if isPublic {
		header += "// Public Endpoint - Auto Generated\n"
	} else {
		header += "// Private Endpoint - Auto Generated\n"
	}

	nuxtIt := `%s
import {Iterator,Pagination} from "assets/iterator.js"

 function GetConfig(baseURL = "http://localhost:3000" ,data={},pagination=null){
     let params = {}
     if (pagination === null){
         pagination = new Pagination({})
     }
     params["items_per_page"] = pagination.ItemsPerPage
     params["page"] = pagination.CurrentPage


     const mergedParams  = { ...params, ...data.params };
     return {
         server: false,
         method: data.method ?? "GET",
         credentials: "include",
         params: mergedParams,
         baseURL: baseURL,
         path: data.path ?? "/"
     }
 }

export default defineNuxtPlugin((nuxtApp) => {
	const url = nuxtApp.$config.public.%s
	const api = {
	%s
	}
    return {
        provide: {
            %s: api,
        },
    };
})`
	n := ToSnakeCase(group)
	_, err = f.WriteString(fmt.Sprintf(nuxtIt, header, n, strings.Join(code, ",\n")+"\n", n))
	return err
}

type Imports struct {
	Name string
	Path string
}

func FormatImports(language Language, list ...Imports) string {
	var output []string
	switch language {
	case LanguageJS:
	case LanguageGo:
		list = append(list, defaultImports...)

	}

	// Sort the list of imports by their Path field
	sort.Slice(list, func(i, j int) bool {
		return list[i].Path < list[j].Path
	})
	dup := map[string]struct{}{}
	for _, i := range list {
		if _, f := dup[i.Path]; f {
			continue
		}
		if i.Path == "" {
			continue
		}
		dup[i.Path] = struct{}{}
		if strings.HasSuffix(i.Path, i.Name) || i.Name == "" {
			output = append(output, fmt.Sprintf(`"%s"`, i.Path))
			continue
		}
		output = append(output, fmt.Sprintf(`%s "%s"`, i.Name, i.Path))
	}
	return strings.Join(output, "\n\t")
}

// groupEndpointsByGroup Helper: Group endpoints by the 'Group' field
func groupEndpointsByGroup(eps []*endpoints.Endpoint) map[string][]*endpoints.Endpoint {
	grouped := make(map[string][]*endpoints.Endpoint)
	for _, ep := range eps {
		group := ep.Group
		if group == "" {
			group = GetBaseDir(ep.URLPath)
			//group = "default"
		}
		grouped[group] = append(grouped[group], ep)
	}
	return grouped
}

func GoNewClientFunc(endpoint *endpoints.Endpoint) []*ClientFunc {
	if endpoint.SkipGenerate {
		return nil
	}

	skipPkg := map[string]bool{
		"string":   true,
		"int":      true,
		"int64":    true,
		"[]string": true,
	}

	var output []*ClientFunc
	re := regexp.MustCompile(`{(.*?)}`)

	for _, method := range endpoint.Methods {
		cf := createClientFunc(endpoint, method, re)
		cf.Language = LanguageGo
		populateQueryParams(cf, endpoint.QueryParams)
		setRequestType(cf, endpoint.RequestTypeMap[strings.ToUpper(method)], skipPkg)
		setResponseType(cf, endpoint.ResponseTypeMap[strings.ToUpper(method)], skipPkg)
		formatPath(cf)
		setMethodName(cf, method)
		additionalChecks(cf, endpoint)

		output = append(output, cf)
	}

	return output
}

func createClientFunc(endpoint *endpoints.Endpoint, method string, re *regexp.Regexp) *ClientFunc {
	return &ClientFunc{
		Path:        endpoint.URLPath,
		MuxVars:     re.FindAllString(endpoint.URLPath, -1),
		MethodType:  cases.Title(language.AmericanEnglish).String(strings.ToLower(method)),
		Imports:     make([]Imports, 0),
		QueryParams: make(map[string]string),
	}
}

func populateQueryParams(cf *ClientFunc, queryParams []string) {
	for _, q := range queryParams {
		camelCaseQP := SnakeCaseToCamelCase(ToSnakeCase(q))
		cf.QueryParams[q] = strings.ToLower(camelCaseQP[:1]) + camelCaseQP[1:]
	}
	cf.Name = UrlToName(cf.Path)
}

func setRequestType(cf *ClientFunc, requestType interface{}, skipPkg map[string]bool) {

	switch cf.Language {
	case LanguageJS:
		if requestType == "" {
			return
		}
		if requestType == nil {
			return
		}
		cf.RequestType = getType(requestType)
		normalName := normalizeName(getType(requestType))

		// Handle maps and arrays
		if isMap(requestType) || strings.HasSuffix(normalName, "{}") {
			cf.RequestType = "Object"
		} else if isArray(requestType) {
			cf.RequestType = fmt.Sprintf("array<%s>", cf.RequestType)
		}
		if strings.HasSuffix(normalName, "{}") {
			cf.RequestTypeName = "object" + strconv.Itoa(rand.IntN(6))
		} else {
			cf.RequestTypeName = normalName
		}

		if _, exists := cf.Objects[normalName]; !exists {
			cf.Objects[normalName] = GetObject(requestType)
		}
	case LanguageGo:
		fallthrough
	default:
		if requestType == "" {
			return
		}
		if requestType == nil {
			return
		}
		fullPkg, pkg := path.Split(getTypePkg(requestType))
		typeName := getType(requestType)
		cf.RequestTypeName = formatName(typeName, isMap(requestType))

		if _, found := skipPkg[pkg]; !found && fullPkg != "" {
			cf.Imports = append(cf.Imports, Imports{
				Name: pkg,
				Path: fullPkg + pkg,
			})
		}

		cf.RequestType = getRequestTypeString(requestType, pkg)
	}

}

func setResponseType(cf *ClientFunc, responseType interface{}, skipPkg map[string]bool) {
	switch cf.Language {
	case LanguageJS:
		if responseType == nil {
			cf.Return = "promise"
			return
		}
		fullPkg := getTypePkg(responseType)
		_, pkg := path.Split(fullPkg)
		cf.DataTypeName = getType(responseType)

		// Add import if necessary
		if fullPkg != "" {
			cf.Imports = append(cf.Imports, Imports{
				Name: pkg,
				Path: fullPkg,
			})
		}

		cf.Return = cf.DataTypeName
		cf.UseIterator = true

		if _, found := cf.Objects[cf.Return]; !found {
			cf.Objects[cf.Return] = GetObject(responseType)
		}

	case LanguageGo:
		fallthrough
	default:
		if responseType == "" || responseType == nil {
			cf.Return = "*clientpkg.ResponseData"
			return
		}

		fullPkg, pkg := path.Split(getTypePkg(responseType))
		cf.DataTypeName = getDataTypeName(responseType, pkg, skipPkg)

		if isArray(responseType) || (!skipPkg[pkg] && fullPkg != "") {
			cf.Imports = append(cf.Imports, Imports{
				Name: pkg,
				Path: fullPkg + pkg,
			})
		}

		cf.Return = fmt.Sprintf("*clientpkg.Iterator[%s]", cf.DataTypeName)
		cf.UseIterator = true
	}
}

func formatPath(cf *ClientFunc) {
	cf.RawPath = cf.Path
	for i, original := range cf.MuxVars {
		n := SnakeCaseToCamelCase(regexp.MustCompile(`[{}]`).ReplaceAllString(original, ""))
		cf.MuxVars[i] = strings.ToLower(n[:1]) + n[1:]
		cf.Path = strings.ReplaceAll(cf.Path, original, "%s")
	}

	if len(cf.MuxVars) == 0 {
		cf.Path = fmt.Sprintf(`"%s"`, cf.Path)
	} else {
		cf.Path = fmt.Sprintf(`fmt.Sprintf("%s", %s)`, cf.Path, strings.Join(cf.MuxVars, ", "))
	}
}

func setMethodName(cf *ClientFunc, method string) {
	switch method {
	case http.MethodGet:
		cf.Name = "Get" + cf.Name
	case http.MethodPost:
		cf.Name = "New" + cf.Name
	case http.MethodDelete:
		cf.Name = "Delete" + cf.Name
	case http.MethodPut, http.MethodPatch:
		cf.Name = "Update" + cf.Name
	}
}

func additionalChecks(cf *ClientFunc, endpoint *endpoints.Endpoint) {
	if len(endpoint.Headers) > 0 {
		cf.UsesHeaderParams = true
	}
	if len(endpoint.QueryParams) > 0 {
		cf.UsesQueryParams = true
	}
}

func JSNewClientFunc(projectName string, endpoint *endpoints.Endpoint) []*ClientFunc {
	var output []*ClientFunc
	if endpoint.SkipGenerate {
		return output
	}
	re := regexp.MustCompile(`\{(.*?)\}`)
	for _, method := range endpoint.Methods {
		clientFunc := createBaseClientFunc(projectName, endpoint, method, re)
		clientFunc.Language = LanguageJS
		setQueryParams(clientFunc, endpoint.QueryParams)
		setRequestType(clientFunc, endpoint.RequestTypeMap[strings.ToUpper(method)], nil)
		setResponseType(clientFunc, endpoint.ResponseTypeMap[strings.ToUpper(method)], nil)

		replaceMuxVars(clientFunc)
		finalizeClientFuncPath(clientFunc)
		setHeaderAndQueryFlags(clientFunc, endpoint)

		// Set method-specific names
		setClientFuncName(clientFunc, method)

		output = append(output, clientFunc)
	}

	return output
}

func createBaseClientFunc(projectName string, endpoint *endpoints.Endpoint, method string, re *regexp.Regexp) *ClientFunc {
	return &ClientFunc{
		UrlEnvVarName: SnakeCaseToCamelCase(ToSnakeCase(projectName)),
		Name:          convertPathToFunctionName(endpoint.URLPath),
		Path:          endpoint.URLPath,
		MuxVars:       re.FindAllString(endpoint.URLPath, -1),
		MethodType:    strings.ToUpper(method),
		Imports:       make([]Imports, 0),
		QueryParams:   map[string]string{},
		Objects:       map[string][]string{},
		Async:         endpoint.Async,
	}
}

func setQueryParams(cf *ClientFunc, queryParams []string) {
	for _, q := range queryParams {
		paramName := SnakeCaseToCamelCase(ToSnakeCase(q))
		cf.QueryParams[q] = strings.ToLower(paramName[:1]) + paramName[1:]
	}
}

func replaceMuxVars(cf *ClientFunc) {
	for i, original := range cf.MuxVars {
		paramName := normalizeName(regexp.MustCompile(`[\{\}]`).ReplaceAllString(original, ""))
		cf.MuxVars[i] = paramName
		cf.Path = strings.ReplaceAll(cf.Path, original, fmt.Sprintf("${%s}", paramName))
	}
}

func finalizeClientFuncPath(cf *ClientFunc) {
	if len(cf.MuxVars) == 0 {
		cf.Path = fmt.Sprintf(`"%s"`, cf.Path)
	} else {
		cf.Path = fmt.Sprintf("`%s`", cf.Path)
	}
}

func setHeaderAndQueryFlags(cf *ClientFunc, endpoint *endpoints.Endpoint) {
	cf.UsesHeaderParams = len(endpoint.Headers) > 0
	cf.UsesQueryParams = len(endpoint.QueryParams) > 0
}

func setClientFuncName(cf *ClientFunc, method string) {
	switch method {
	case http.MethodGet:
		cf.Name = "Get" + cf.Name
	case http.MethodPost:
		cf.Name = "New" + cf.Name
	case http.MethodDelete:
		cf.Name = "Delete" + cf.Name
	case http.MethodPut, http.MethodPatch:
		cf.Name = "Update" + cf.Name
	}
}

func normalizeName(name string) string {
	normalized := SnakeCaseToCamelCase(ToSnakeCase(name))
	return strings.ToLower(normalized[:1]) + normalized[1:]
}

func GetObject(i interface{}) []string {
	var o []string
	structType := reflect.TypeOf(i)
	if structType == nil {
		return nil
	}
	if structType.Name() == "" {
		structType = structType.Elem()
	}
	switch i.(type) {
	case string:
		return []string{"string"}
	case []string:
		return []string{"arr_string"}
	case int:
		return []string{"int"}
	case []int:
		return []string{"arr_int"}
	case map[string]string:
		return []string{"string_map"}
	case map[string]interface{}:
		return []string{"string_interface_map"}
	case map[interface{}]interface{}:
		return []string{"interface_interface_map"}
	case map[int64]interface{}:
		return []string{"int64_interface_map"}
	case map[int64]string:
		return []string{"int64_string_map"}
	}
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		name := field.Tag.Get("json")
		if name == "" {
			name = field.Name
		}
		if name == "-" {
			continue
		}
		name = strings.ReplaceAll(name, ",omitempty", "")
		o = append(o, name)
	}
	return o
}

func convertPathToFunctionName(path string) string {
	// Remove leading and trailing slashes
	path = strings.Trim(path, "/")

	// Replace path segments with placeholders
	// For example, /account/{account_id}/user/{user_id}/settings/query becomes accountUserSettingsQuery
	re := regexp.MustCompile(`\{[^}]+\}`)
	path = re.ReplaceAllString(path, "ByID")

	// Replace slashes with capitalized words
	parts := strings.Split(path, "/")
	for i := 0; i < len(parts); i++ {
		// Capitalize the first letter of each segment
		if len(parts[i]) > 0 {
			parts[i] = strings.Title(parts[i])
		}
	}

	// Join the segments into a single function name
	return strings.Join(parts, "")
}
