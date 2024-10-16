package generators

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
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

func getTypePkg(myVar interface{}) (string, string) {
	switch myVar.(type) {
	case string:
		return "string", ""
	case int64:
		return "int64", ""
	case []string:
		return "[]string", ""
	}

	t := reflect.TypeOf(myVar)
	if t == nil {
		return "", ""
	}

	var pkgPath string
	if isArray(myVar) {
		pkgPath = t.Elem().PkgPath()
	} else if t.Kind() == reflect.Ptr {
		pkgPath = t.Elem().PkgPath()
	} else {
		pkgPath = t.PkgPath()
	}

	// Handle versioning removal, if the package ends with /v[0-9]+
	re := regexp.MustCompile(`/v[0-9]+$`)
	basePkgPath := re.ReplaceAllString(pkgPath, "")

	// Handle hyphen case and extract the first segment
	segments := strings.Split(basePkgPath, "/")
	lastSegment := segments[len(segments)-1]
	if strings.Contains(lastSegment, "-") {
		// Return the first part before the hyphen
		firstPart := strings.Split(lastSegment, "-")[0]
		return pkgPath, firstPart
	}

	return pkgPath, lastSegment
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
func writeToGoFile(dir string, group string, code []string, isPublic bool, overrides map[string]*Imports, imports ...Imports) error {
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

`, ToSnakeCase(pkg), FormatImports(LanguageGo, overrides, imports...))
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
func writeNuxtFile(dir string, group string, code []string, isPublic bool, classes map[string][]string, imports ...Imports) error {
	if len(code) == 0 {
		return nil
	}
	filename := fmt.Sprintf("%s_plugin.js", ToSnakeCase(group))
	fp := filepath.Join(dir, filename)
	classFilename := fmt.Sprintf("%s_classes.ts", ToSnakeCase(group))
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()
	var classNames []string
	for c, _ := range classes {
		classNames = append(classNames, c)
	}

	_, pkg := path.Split(dir)
	header := fmt.Sprintf(
		` // %s
// AUTO GENERATED
%s


`, ToSnakeCase(pkg), FormatImports(LanguageJS, nil, imports...))
	// Add package name and imports at the top of the file
	if isPublic {
		header += "// Public Endpoint - Auto Generated\n"
	} else {
		header += "// Private Endpoint - Auto Generated\n"
	}
	classImports := fmt.Sprintf(`import {%s} from "./%s"`, strings.Join(classNames, ","), classFilename)
	if len(classNames) == 0 {
		classImports = ""
	}
	nuxtIt := `%s
import {Iterator,Pagination} from "./iterator.js"
%s
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
	_, err = f.WriteString(fmt.Sprintf(nuxtIt, classImports, header, n, strings.Join(code, ",\n")+"\n", n))
	return err
}

// writeNuxtFile Helper: Write generated function code to a file
func writeClassFile(dir string, group string, code string, isPublic bool, imports ...Imports) error {
	if len(code) == 0 {
		return nil
	}
	filename := fmt.Sprintf("%s_classes.ts", ToSnakeCase(group))
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


`, ToSnakeCase(pkg), FormatImports(LanguageJS, nil, imports...))
	// Add package name and imports at the top of the file
	if isPublic {
		header += "// Public Endpoint - Auto Generated\n"
	} else {
		header += "// Private Endpoint - Auto Generated\n"
	}
	_, err = f.WriteString(code)
	return err
}

type Imports struct {
	Name string
	Path string
}

func FormatImports(language Language, overrides map[string]*Imports, list ...Imports) string {
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
		if override, ok := overrides[i.Path]; ok {
			i.Name = override.Name
		}
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
		Description: endpoint.Description,
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
		switch reflect.TypeOf(requestType).Kind() {
		case reflect.Struct:
			fallthrough
		case reflect.Ptr:
			if _, exists := cf.Objects[strings.ToTitle(normalName[:1])+normalName[1:]]; !exists {
				cf.Objects[strings.ToTitle(normalName[:1])+normalName[1:]] = GetObject(requestType)
			}
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
		fullPkg, pkg := getTypePkg(requestType)
		typeName := getType(requestType)

		cf.RequestTypeName = formatName(typeName, isMap(requestType))

		if _, found := skipPkg[pkg]; !found && fullPkg != "" {
			cf.Imports = append(cf.Imports, Imports{
				Name: pkg,
				Path: fullPkg,
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
		fullPkg, pkg := getTypePkg(responseType)
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
		switch reflect.TypeOf(responseType).Kind() {
		case reflect.Struct:
			fallthrough
		case reflect.Ptr:
			if _, exists := cf.Objects[strings.ToTitle(cf.Return[:1])+cf.Return[1:]]; !exists {
				cf.Objects[strings.ToTitle(cf.Return[:1])+cf.Return[1:]] = GetObject(responseType)
			}
		}

	case LanguageGo:
		fallthrough
	default:
		if responseType == "" || responseType == nil {
			cf.Return = "*clientpkg.ResponseData"
			return
		}
		if strings.Contains(cf.Path, "{") {
			cf.Imports = append(cf.Imports, Imports{
				Name: "fmt",
				Path: "fmt",
			})
		}
		fullPkg, pkg := getTypePkg(responseType)
		cf.DataTypeName = getDataTypeName(responseType, pkg, skipPkg)

		if isArray(responseType) || (!skipPkg[pkg] && fullPkg != "") {
			cf.Imports = append(cf.Imports, Imports{
				Name: pkg,
				Path: fullPkg,
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
		if strings.Contains(name, ",") {
			continue
		}
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
			parts[i] = strings.ToTitle(parts[i])
		}
	}

	// Join the segments into a single function name
	return strings.Join(parts, "")
}

func GetGoFiles(path string) []string {
	libRegEx, e := regexp.Compile(`^.+\.(go)$`)
	if e != nil {
		log.Fatal(e)
	}
	var files []string
	_ = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, "vendor") {
			return nil
		}
		if err == nil && libRegEx.MatchString(info.Name()) {
			files = append(files, path)
		}
		return nil
	})
	return files
}

type Func struct {
	File     string
	Name     string
	Ln       int
	Comment  *Comment
	Data     string
	Override map[string]string
}

func (fc *Func) UpdateComment() error {
	f, err := os.Open(fc.File)
	if err != nil {
		return err
	}
	defer f.Close()
	var lines []string
	// Splits on newlines by default.
	scanner := bufio.NewScanner(f)
	line := 1
	if fc.Comment.Start == 0 {
		fc.Comment.Start = fc.Ln - 1
		fc.Comment.End = fc.Ln - 1
	}
	commentRegex := regexp.MustCompile(`//|/\*|\*/`)
	dontSkipped := false
	wrote := false
	for scanner.Scan() {
		text := scanner.Text()
		if line >= fc.Comment.Start && line <= fc.Comment.End && !wrote {
			for i, l := range fc.Comment.Lines {
				if i == 0 && !(strings.HasPrefix(l, "//") || strings.HasPrefix(l, "/*")) {
					l = `/*` + l
					dontSkipped = true
				}

				if i >= len(fc.Comment.Lines)-1 && dontSkipped {
					l += "*/"
				}
				lines = append(lines, l)
			}
			wrote = true

		} else if line >= fc.Comment.Start && line <= fc.Comment.End {

		} else {
			lines = append(lines, text)
		}
		if line == fc.Comment.End && !commentRegex.MatchString(text) && text != "" {
			lines = append(lines, text)
		}
		line++
	}
	_ = f.Close()

	err = os.WriteFile(fc.File, []byte(strings.Join(lines, "\n")), os.ModePerm)
	return err
}

type Comment struct {
	Start int
	End   int
	Lines []string
}

func (c *Comment) Set(cmt string) {
	c.Lines = strings.Split(cmt, "\n")
}

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
func FindFunction(fName string, goFiles []string) map[string]Func {
	found := map[string]Func{}
	for _, files := range goFiles {
		if cmt, ln, d, err := FindString(files, regexp.MustCompile(`func[\(\s\*a-z]*`+fName+`\s{0,1}\(`)); err == nil && ln > 0 {
			found[files] = Func{
				File:    files,
				Name:    fName,
				Ln:      ln,
				Comment: cmt,
				Data:    d,
			}
		}
		//} else if cmt, ln, err := FindString(files, regexp.MustCompile(`func[\(\)\s\*a-zA-Z\[\]]*`+fName+`\s{0,1}\(`)); err == nil && ln > 0 {
		//	found[files] = Func{
		//		File:    files,
		//		Name:    fName,
		//		Ln:      ln,
		//		Comment: cmt,
		//	}
		//}

	}
	return found
}

func FindString(file string, find *regexp.Regexp) (*Comment, int, string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, 0, "", err
	}
	defer f.Close()

	// Splits on newlines by default.
	scanner := bufio.NewScanner(f)

	line := 1
	// https://golang.org/pkg/bufio/#Scanner.Scan
	comment := &Comment{
		Lines: []string{},
	}
	startComment := false
	doubleSlash := false
	commentRegex := regexp.MustCompile(`//|/\*|\*/`)
	funcRegex := regexp.MustCompile(`func[\(\)\s\*a-zA-Z\[\]]\s{0,1}`)
	var functionCode []string
	inFunction := false
	foundFunc := false
	//funcStartLine := 0
	//functionBreaks :=0
	functionLine := 0
	for scanner.Scan() {
		text := scanner.Text()

		// Detect the start of a function
		if funcRegex.MatchString(text) {
			if functionLine == 0 {
				functionLine = line
			}
			if inFunction {
				// Handle nested functions or situations where multiple functions are present
				functionCode = append(functionCode, text)
			} else {
				inFunction = true
				//funcStartLine = line
				functionCode = []string{text}

			}
		} else if inFunction {
			// Append lines while inside a function
			functionCode = append(functionCode, text)
			if strings.Contains(text, "}") {
				inFunction = false
				if foundFunc {
					return comment, functionLine, strings.Join(functionCode, "\n"), nil
				}
			}
		}

		if (strings.Contains(text, "/*") || strings.HasPrefix(text, "//")) && !startComment {
			startComment = true
			comment.Start = line
			comment.Lines = []string{}
			doubleSlash = strings.Contains(text, "//")
		}
		if doubleSlash && startComment && !strings.HasPrefix(text, "//") && !(strings.TrimSpace(text) == "" || text == "\n") {
			startComment = false
		}
		if startComment {
			comment.Lines = append(comment.Lines, strings.TrimSpace(commentRegex.ReplaceAllString(text, "")))
		}

		if strings.Contains(text, "*/") {
			startComment = false
			comment.End = line
		}

		if find.MatchString(text) {
			comment.End = line - 1
			foundFunc = true
			continue
			//return comment, line, strings.Join(functionCode, "\n"), nil
			//return comment, line, strings.Join(functionCode, "\n"), nil
		}
		if funcRegex.MatchString(text) || strings.HasPrefix(text, "import") {
			startComment = false
			comment.Start = 0
			comment.End = 0
			doubleSlash = false
		}
		line++
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, "", err
	}
	return nil, 0, "", nil
}
