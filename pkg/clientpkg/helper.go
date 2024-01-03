package clientpkg

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/Seann-Moser/go-serve/server/endpoints"
)

//go:embed templates/go_function_template.tmpl
var functionTemplate string

//go:embed templates/struct_template.tmpl
var startingTemplate string

//go:embed templates/js_function_template.tmpl
var jsFunctionTemplate string

//go:embed templates/js_classes.tmpl
var jsClassesTemplate string

//go:embed templates/iterator.js
var jsIterator string

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func GetFlagWithPrefix(prefix, flag string) string {
	if prefix == "" {
		return flag
	}
	return fmt.Sprintf("%s-%s", prefix, flag)
}

func MergeMap[T any](m1, m2 map[string]T) map[string]T {
	if m1 == nil {
		return m2
	}
	if m2 == nil {
		return m1
	}
	if m1 == nil && m2 == nil {
		return map[string]T{}
	}
	for k, v := range m2 {
		if _, found := m1[k]; found {
			continue
		}
		m1[k] = v
	}
	return m1
}
func GetProjectDir() (string, string, error) {
	currentPath, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	homeDir = path.Join(homeDir, "go", "src") + "/"

	rootDir := ""
	count := 0

	for _, i := range strings.Split(strings.ReplaceAll(currentPath, homeDir, ""), "/") {
		rootDir = path.Join(rootDir, i)
		if count > 1 {

			break
		}
		count++
	}
	return path.Join(homeDir, rootDir), rootDir, nil
}

func GenerateBaseClient(write bool, headers []string, endpoints ...*endpoints.Endpoint) (string, error) {
	currentPath, err := os.Getwd()
	if err != nil {
		return "", err
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	homeDir = path.Join(homeDir, "go", "src") + "/"

	rootDir := ""
	count := 0

	for _, i := range strings.Split(strings.ReplaceAll(currentPath, homeDir, ""), "/") {
		rootDir = path.Join(rootDir, i)
		if count > 1 {

			break
		}
		count++
	}
	_, projectName := path.Split(rootDir)
	envVarName := snakeCaseToCamelCase(ToSnakeCase(GetFlagWithPrefix("base-url", projectName)))
	envVarName = strings.ToLower(envVarName[:1]) + envVarName[1:]
	var functions []string
	var imports []string
	var jsFunctions []string

	var objects map[string][]string
	imports = append(imports, []string{
		`"context"`,
		`"fmt"`,
		`"net/http"`,
		`"github.com/spf13/pflag"`,
		`clientpkg "github.com/Seann-Moser/go-serve/pkg/clientpkg"`,
		`"github.com/spf13/viper"`,
		`"strings"`,
		`"github.com/Seann-Moser/go-serve/pkg/response"`,
	}...)

	for _, e := range endpoints {
		cfList := GoNewClientFunc(e)
		for _, cf := range cfList {
			output, err := templateReplaceData(functionTemplate, cf)
			if err != nil {
				return "", err
			}
			imports = append(imports, cf.Imports...)

			functions = append(functions, output)
		}
		cfList = JSNewClientFunc(envVarName, e)
		for _, cf := range cfList {
			output, err := templateReplaceData(jsFunctionTemplate, cf)
			if err != nil {
				return "", err
			}
			jsFunctions = append(jsFunctions, output)
			objects = MergeMap[[]string](cf.Objects, objects)
		}

	}

	class, err := templateReplaceClasses(jsClassesTemplate, objects)
	if err != nil {
		return "", err
	}
	imports = RemoveDuplicateValues[string](imports)

	pkgName := fmt.Sprintf("%s_client", ToSnakeCase(projectName))
	clientDir := fmt.Sprintf("pkg/%s", pkgName)
	clientDir = path.Join(homeDir, rootDir, clientDir)
	//importPath := strings.ReplaceAll(currentPath, homeDir, "")
	err = createDir(clientDir)
	if err != nil {
		return "", err
	}
	for i := range headers {
		headers[i] = strings.ReplaceAll(ToSnakeCase(strings.ReplaceAll(headers[i], "-", "_")), "_", "-")
	}
	starting, err := templateReplaceGenerate(startingTemplate,
		map[string]interface{}{
			"Name":    projectName,
			"Package": pkgName,
			"Imports": imports,
			"Headers": headers,
			//"Args": args
		})
	if err != nil {
		return "", err
	}
	functions = append([]string{starting}, functions...)
	jsFunctions =
		[]string{jsIterator, class, fmt.Sprintf(`
export default defineNuxtPlugin((nuxtApp) => {
	const api = {
	%s
	}
    return {
        provide: {
            %s: api,
        },
    };
})
`, strings.Join(jsFunctions, ","), ToSnakeCase(projectName))}

	if write {
		err = os.WriteFile(path.Join(clientDir, fmt.Sprintf("generated_%s.go", ToSnakeCase(projectName))), []byte(strings.Join(functions, "")), os.ModePerm)
		if err != nil {
			return "", err
		}

		err = os.WriteFile(path.Join(clientDir, fmt.Sprintf("generated_%s.js", ToSnakeCase(projectName))), []byte(strings.Join(jsFunctions, "")), os.ModePerm)
		if err != nil {
			return "", err
		}
	}
	return strings.Join(functions, ""), nil
}

type ClientFunc struct {
	Name          string
	UrlEnvVarName string
	Return        string

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

	Imports []string

	Objects map[string][]string
}

func JSNewClientFunc(projectName string, endpoint *endpoints.Endpoint) []*ClientFunc {
	var output []*ClientFunc
	if endpoint.SkipGenerate {
		return output
	}
	for _, m := range endpoint.Methods {
		re := regexp.MustCompile(`\{(.*?)\}`)
		cf := &ClientFunc{
			UrlEnvVarName: projectName,
			Path:          endpoint.URLPath,
			MuxVars:       re.FindAllString(endpoint.URLPath, -1),
			MethodType:    strings.ToUpper(m),
			Imports:       make([]string, 0),
			QueryParams:   map[string]string{},
			Objects:       map[string][]string{},
			Async:         endpoint.Async,
		}
		for _, q := range endpoint.QueryParams {
			cf.QueryParams[q] = snakeCaseToCamelCase(ToSnakeCase(q))
			cf.QueryParams[q] = strings.ToLower(cf.QueryParams[q][:1]) + cf.QueryParams[q][1:]
		}
		cf.Name = UrlToName(cf.Path)

		if requestType, found := endpoint.RequestTypeMap[strings.ToUpper(m)]; found {
			cf.RequestType = getType(requestType)
			normalName := snakeCaseToCamelCase(ToSnakeCase(getType(requestType)))
			n := strings.ToLower(normalName[:1]) + normalName[1:]
			if isMap(requestType) {
				cf.RequestType = "Object"
			} else if isArray(requestType) {
				cf.RequestType = fmt.Sprintf("array<%s>", cf.RequestType)
			}
			cf.RequestTypeName = n

			//cf.Imports = append(cf.Imports, fmt.Sprintf(`import %s from "%s"`, pkg, fullPkg))
			if _, found := cf.Objects[normalName]; !found {
				cf.Objects[normalName] = GetObject(requestType)
			}
		}
		if responseType, found := endpoint.ResponseTypeMap[strings.ToUpper(m)]; found {
			fullPkg := getTypePkg(responseType)
			_, pkg := path.Split(fullPkg)
			cf.DataTypeName = getType(responseType)
			if fullPkg != "" {
				cf.Imports = append(cf.Imports, fmt.Sprintf(`%s "%s"`, pkg, fullPkg))

			}

			cf.Return = strings.Join([]string{cf.DataTypeName}, ",")
			cf.UseIterator = true
			if _, found := cf.Objects[cf.Return]; !found {
				cf.Objects[cf.Return] = GetObject(responseType)
			}
		} else {
			cf.Return = "promise"
		}

		for i := range cf.MuxVars {
			original := cf.MuxVars[i]
			n := snakeCaseToCamelCase(regexp.MustCompile(`[\{\}]`).ReplaceAllString(cf.MuxVars[i], ""))
			n = strings.ToLower(n[:1]) + n[1:]
			cf.MuxVars[i] = n
			cf.Path = strings.ReplaceAll(cf.Path, original, fmt.Sprintf(`${%s}`, n))
		}

		if len(cf.MuxVars) == 0 {
			cf.Path = fmt.Sprintf(`"%s"`, cf.Path)
		} else {
			cf.Path = fmt.Sprintf("`%s`", cf.Path)
		}
		if len(endpoint.Headers) > 0 {
			cf.UsesHeaderParams = true
		}
		if len(endpoint.QueryParams) > 0 {
			cf.UsesQueryParams = true
		}
		switch m {
		case http.MethodGet:
			cf.Name = "Get" + cf.Name
		case http.MethodPost:
			cf.Name = "New" + cf.Name
		case http.MethodDelete:
			cf.Name = "Delete" + cf.Name
		//case http.MethodPatch:
		//	cf.Name = "Update"+cf.Name
		case http.MethodPut:
			cf.Name = "Update" + cf.Name
		default:
			continue
		}

		output = append(output, cf)
	}

	return output
}

func GetObject(i interface{}) []string {
	var o []string
	structType := reflect.TypeOf(i)
	if structType.Name() == "" {
		structType = structType.Elem()
	}
	switch i.(type) {
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
func GoNewClientFunc(endpoint *endpoints.Endpoint) []*ClientFunc {
	var output []*ClientFunc
	if endpoint.SkipGenerate {
		return output
	}
	for _, m := range endpoint.Methods {
		re := regexp.MustCompile(`\{(.*?)\}`)
		cf := &ClientFunc{
			Path:        endpoint.URLPath,
			MuxVars:     re.FindAllString(endpoint.URLPath, -1),
			MethodType:  cases.Title(language.AmericanEnglish).String(strings.ToLower(m)),
			Imports:     make([]string, 0),
			QueryParams: map[string]string{},
		}
		for _, q := range endpoint.QueryParams {
			cf.QueryParams[q] = snakeCaseToCamelCase(ToSnakeCase(q))
			cf.QueryParams[q] = strings.ToLower(cf.QueryParams[q][:1]) + cf.QueryParams[q][1:]
		}
		cf.Name = UrlToName(cf.Path)

		if requestType, found := endpoint.RequestTypeMap[strings.ToUpper(m)]; found {
			fullPkg := getTypePkg(requestType)
			_, pkg := path.Split(fullPkg)
			if isMap(requestType) {
				cf.RequestType = fmt.Sprintf("map[%s]%s", getType(requestType), getType(requestType))
			} else if isArray(requestType) {
				cf.RequestType = fmt.Sprintf("[]*%s.%s", pkg, getType(requestType))
			} else {
				cf.RequestType = fmt.Sprintf("*%s.%s", pkg, getType(requestType))
			}

			n := snakeCaseToCamelCase(ToSnakeCase(getType(requestType)))
			if len(n) == 0 {
				log.Fatal("unable to get name for requesttype")
			}
			if isMap(requestType) {
				n += "Map"
			}
			n = strings.ToLower(n[:1]) + n[1:]
			if fullPkg != "" {
				cf.Imports = append(cf.Imports, fmt.Sprintf(`%s "%s"`, pkg, fullPkg))
			}

			cf.RequestTypeName = n

		}
		if responseType, found := endpoint.ResponseTypeMap[strings.ToUpper(m)]; found {
			fullPkg := getTypePkg(responseType)
			_, pkg := path.Split(fullPkg)
			cf.DataTypeName = fmt.Sprintf("%s.%s", pkg, getType(responseType))
			cf.Imports = append(cf.Imports, fmt.Sprintf(`%s "%s"`, pkg, fullPkg))

			cf.Return = strings.Join([]string{fmt.Sprintf("*clientpkg.Iterator[%s]", cf.DataTypeName)}, ",")
			cf.UseIterator = true
		} else {
			cf.Return = "*clientpkg.ResponseData"
		}

		for i := range cf.MuxVars {
			original := cf.MuxVars[i]
			n := snakeCaseToCamelCase(regexp.MustCompile(`[\{\}]`).ReplaceAllString(cf.MuxVars[i], ""))
			n = strings.ToLower(n[:1]) + n[1:]
			cf.MuxVars[i] = n
			cf.Path = strings.ReplaceAll(cf.Path, original, "%s")
		}

		if len(cf.MuxVars) == 0 {
			cf.Path = fmt.Sprintf(`"%s"`, cf.Path)
		} else {
			cf.Path = fmt.Sprintf(`fmt.Sprintf("%s", %s)`, cf.Path, strings.Join(cf.MuxVars, ", "))
		}
		if len(endpoint.Headers) > 0 {
			cf.UsesHeaderParams = true
		}
		if len(endpoint.QueryParams) > 0 {
			cf.UsesQueryParams = true
		}
		switch m {
		case http.MethodGet:
			cf.Name = "Get" + cf.Name
		case http.MethodPost:
			cf.Name = "New" + cf.Name
		case http.MethodDelete:
			cf.Name = "Delete" + cf.Name
		//case http.MethodPatch:
		//	cf.Name = "Update"+cf.Name
		case http.MethodPut:
			cf.Name = "Update" + cf.Name
		default:
			continue
		}

		output = append(output, cf)
	}

	return output
}

func StringArray(key string, count int) []string {
	var output []string
	for i := 0; i < count; i++ {
		output = append(output, key)
	}
	return output
}

func UrlToName(url string) string {
	re := regexp.MustCompile(`\{(.*?)\}`)
	for _, d := range re.FindAllString(url, -1) {
		url = strings.ReplaceAll(url, "/"+d, "")

	}
	url = strings.TrimPrefix(url, "/")
	url = strings.TrimSuffix(url, "/")

	url = strings.ReplaceAll(url, "/", "_")
	return snakeCaseToCamelCase(url)
}

func RemoveDuplicateValues[T comparable](intSlice []T) []T {
	keys := make(map[T]bool)
	var list []T
	for _, entry := range intSlice {
		var t T
		if entry == t {
			continue
		}
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func templateReplaceClasses(rawTmpl string, data map[string][]string) (string, error) {
	tmpl, err := template.New("general").Parse(rawTmpl)
	if err != nil {
		panic(err)
	}
	buff := bytes.NewBufferString("")
	err = tmpl.Execute(buff, data)
	if err != nil {
		panic(err)
	}
	return buff.String(), nil
}

func templateReplaceGenerate(rawTmpl string, data map[string]interface{}) (string, error) {
	tmpl, err := template.New("general").Parse(rawTmpl)
	if err != nil {
		panic(err)
	}
	buff := bytes.NewBufferString("")
	err = tmpl.Execute(buff, data)
	if err != nil {
		panic(err)
	}
	return buff.String(), nil
}
func templateReplaceData(rawTmpl string, data *ClientFunc) (string, error) {
	tmpl, err := template.New(data.Name).Parse(rawTmpl)
	if err != nil {
		return "", err
	}
	buff := bytes.NewBufferString("")
	err = tmpl.Execute(buff, data)
	if err != nil {
		panic(err)
	}
	return buff.String(), nil
}

func templateReplace(rawTmpl string, data interface{}) (string, error) {
	tmpl, err := template.New("general").Parse(rawTmpl)
	if err != nil {
		return "", err
	}
	buff := bytes.NewBufferString("")
	err = tmpl.Execute(buff, data)
	if err != nil {
		panic(err)
	}
	return buff.String(), nil
}
func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
func getTypePkg(myVar interface{}) string {
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
	if t.Kind() == reflect.Ptr {
		return isArray(t.Elem())
	} else {
		return t.Name() == ""
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
	t := reflect.TypeOf(myVar)
	if t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		if t.Name() == "" {
			return t.Elem().Name()
		}
		return t.Name()
	}
}

func ToSnakeCase(str string) string {
	str = strings.ReplaceAll(str, "-", "_")
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(strings.ReplaceAll(snake, "__", "_"))
}

func createDir(dir string) error {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func snakeCaseToHeader(inputUnderScoreStr string) (camelCase string) {
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
					camelCase += "-"
				} else {
					camelCase += string(v)
				}
			}
		}
	}
	return

}

func snakeCaseToCamelCase(inputUnderScoreStr string) (camelCase string) {
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

// HandlerFuncs godoc
// @Summary todo
// @Tags 
// @ID account_user_settings-GET-DELETE
// @Produce json 
// @Param account_id path string true "todo" 
// @Param user_id path string true "todo" 
// @Param header header string false "todo" 
// @Param test header string false "todo" 
// @Param responseData body clientpkg.ResponseData false "todo" 
// @Success 200 {object} response.BaseResponse "todo"  
// @Failure 400 {object} response.BaseResponse "todo"
// @Failure 500 {object} response.BaseResponse "todo"
// @Failure 401 {object} response.BaseResponse "todo"
// @Router /account/{account_id}/user/{user_id}/settings [GET] 
// @Router /account/{account_id}/user/{user_id}/settings [DELETE] 
func (c *Client) HandlerFuncs(w http.ResponseWriter, r *http.Request) {

}

// HandlerFuncs godoc
// @Summary todo
// @Tags 
// @ID account_user-GET
// @Produce json 
// @Param account_id path string true "todo" 
// @Param user_id path string true "todo" 
// @Success 200 {object} response.BaseResponse{data=clientpkg.RequestData} "todo"  
// @Failure 400 {object} response.BaseResponse "todo"
// @Failure 500 {object} response.BaseResponse "todo"
// @Failure 401 {object} response.BaseResponse "todo"
// @Router /account/{account_id}/user/{user_id} [GET] 
func HandlerFuncs(w http.ResponseWriter, r *http.Request) {

}