package clientpkg

import (
	_ "embed"
	"fmt"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	"os"
	"path"
	"regexp"
	"strings"
)

//go:embed templates/js_function_template.tmpl
var jsFunctionTemplate string

//go:embed templates/js_classes.tmpl
var jsClassesTemplate string

//go:embed templates/iterator.js
var jsIterator string

//go:embed templates/js_func.tmpl
var jsFuncTmpl string

func GenerateBaseJSClient(write bool, headers []string, endpoints ...*endpoints.Endpoint) (string, error) {
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
	_, projectName := path.Split(rootDir)
	envVarName := snakeCaseToCamelCase(ToSnakeCase(GetFlagWithPrefix("base-url", projectName)))
	envVarName = strings.ToLower(envVarName[:1]) + envVarName[1:]

	var jsFunctions []string
	var objects map[string][]string
	groupings := map[string][]string{}

	for _, e := range endpoints {
		if e == nil {
			continue
		}
		groupSplit := strings.Split(e.URLPath, "/")
		group := ""
		if len(groupSplit) > 1 {
			group = groupSplit[1]

		}
		cfList := JSNewClientFunc(envVarName, e)
		for _, cf := range cfList {
			output, err := templateReplaceData(jsFuncTmpl, cf)
			if err != nil {
				return "", err
			}
			if _, found := groupings[group]; !found {
				groupings[group] = []string{}
			}
			jsFunctions = append(jsFunctions, output)
			objects = MergeMap[[]string](cf.Objects, objects)
			groupings[group] = append(groupings[group], output)
		}

	}

	class, err := templateReplaceClasses(jsClassesTemplate, objects)
	if err != nil {
		return "", err
	}
	pkgName := fmt.Sprintf("%s_client", ToSnakeCase(projectName))

	clientDir := path.Join("pkg", pkgName)
	clientDir = path.Join(homeDir, rootDir, clientDir)
	err = createDir(clientDir)
	if err != nil {
		return "", err
	}

	if write {
		err = os.WriteFile(path.Join(clientDir, fmt.Sprintf("%s_assets.js", ToSnakeCase(projectName))), []byte(class), os.ModePerm)
		if err != nil {
			return "", err
		}
	}
	for k, funcs := range groupings {
		jsFunctionsStr := `
import {Iterator,Pagination} from "assets/iterator.js"
const runtimeConfig = useRuntimeConfig();

` + strings.Join(funcs, "\n\n")
		err = os.WriteFile(path.Join(clientDir, fmt.Sprintf("%s_%s_req.js", ToSnakeCase(projectName), ToSnakeCase(k))), []byte(jsFunctionsStr), os.ModePerm)
		if err != nil {
			return "", err
		}

	}
	return "", nil
}
