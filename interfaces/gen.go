package interfaces

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
)

//go:embed handlerTemplate.txt
var handlerTemplate string

func GenerateHTTPHandlers(interfaceSrc, packageName, outputDir string) error {
	// Parse the interface source code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "interface.go", interfaceSrc, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse interface source: %v", err)
	}

	var iface *ast.InterfaceType
	var ifaceName string

	// Find the interface in the AST
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Type == nil {
				continue
			}
			if astType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				iface = astType
				ifaceName = typeSpec.Name.Name
				break
			}
		}
		if iface != nil {
			break
		}
	}

	if iface == nil {
		return fmt.Errorf("no interface found in the provided source")
	}

	var methods []Method

	for _, method := range iface.Methods.List {
		// Each method can have multiple names (unlikely in interfaces, but possible)
		for _, methodName := range method.Names {
			funcType, ok := method.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			// Extract parameters
			var params []Param
			var queryParams []string
			var muxVar []string
			if funcType.Params != nil {
				for i, param := range funcType.Params.List {
					var paramName string
					if len(param.Names) > 0 {
						paramName = param.Names[0].Name
					} else {
						// Generate a parameter name if not provided
						paramName = fmt.Sprintf("param%d", i)
					}
					if strings.HasSuffix(strings.ToLower(paramName), "id") {
						muxVar = append(muxVar, paramName)
					}
					paramType := exprToString(param.Type)
					// Skip context.Context parameter
					if paramType == "context.Context" && paramName == "ctx" {
						continue
					}
					params = append(params, Param{
						Name: paramName,
						Type: paramType,
					})
					queryParams = append(queryParams, paramName)
				}
			}

			// Extract return types
			var returns []Return
			if funcType.Results != nil {
				for _, result := range funcType.Results.List {
					resultType := exprToString(result.Type)
					returns = append(returns, Return{
						Type: resultType,
					})
				}
			}

			// Determine HTTP method based on the function name
			httpMethod := determineHTTPMethod(methodName.Name)

			// Determine URL path
			urlPath := "/" + strings.ToLower(ifaceName) + "/" + toSnakeCase(methodName.Name)

			// Determine ResponseTypeMap and RequestTypeMap
			var responseType string
			var requestType string

			// Handle ResponseType
			if len(returns) > 1 {
				// Assuming the last return type is error
				responseType = returns[0].Type
			} else if len(returns) == 1 {
				if returns[0].Type != "error" {
					responseType = returns[0].Type
				} else {
					responseType = ""
				}
			} else {
				responseType = ""
			}

			// Handle RequestType
			if len(params) > 0 {
				// If multiple params, define a struct type
				if len(params) == 1 {
					requestType = params[0].Type
				} else {
					requestType = fmt.Sprintf("%sRequest", methodName.Name)
				}
			} else {
				requestType = ""
			}

			methods = append(methods, Method{
				Name:         methodName.Name,
				HTTPMethod:   httpMethod,
				HandlerName:  fmt.Sprintf("%sHandler", methodName.Name),
				URLPath:      urlPath,
				Params:       params,
				Returns:      returns,
				QueryParams:  queryParams,
				ResponseType: responseType,
				RequestType:  requestType,
			})
		}
	}

	// Prepare the template for the handler file

	// Create a FuncMap for template functions
	funcMap := template.FuncMap{
		"toSnakeCase":  toSnakeCase,
		"or":           orFunc,
		"title":        strings.Title,
		"toLower":      strings.ToLower,
		"hasError":     hasError,
		"hasOnlyError": hasOnlyError,
		"hasMultiple":  hasMultiple,
	}

	// Parse the template
	tmpl, err := template.New("handler").Funcs(funcMap).Parse(handlerTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Prepare data for the template
	data := struct {
		PackageName   string
		InterfaceName string
		Methods       []Method
	}{
		PackageName:   packageName,
		InterfaceName: ifaceName,
		Methods:       methods,
	}

	// Execute the template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	// Ensure the output directory exists
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Write to a Go file
	outputFile := fmt.Sprintf("%s/handlers.go", outputDir)
	err = ioutil.WriteFile(outputFile, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write handler file: %v", err)
	}

	log.Printf("HTTP handlers generated successfully at %s", outputFile)
	return nil
}

// Helper function to convert AST expressions to string
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	default:
		return ""
	}
}

// determineHTTPMethod infers the HTTP method based on the function name.
func determineHTTPMethod(funcName string) string {
	// Simple heuristic: CRUD operations
	switch {
	case strings.HasPrefix(funcName, "Get") || strings.HasPrefix(funcName, "List"):
		return "GET"
	case strings.HasPrefix(funcName, "Create") || strings.HasPrefix(funcName, "New"):
		return "POST"
	case strings.HasPrefix(funcName, "Update"):
		return "PUT"
	case strings.HasPrefix(funcName, "Delete") || strings.HasPrefix(funcName, "Remove"):
		return "DELETE"
	default:
		return "POST" // default to POST
	}
}

// toSnakeCase converts a CamelCase string to snake_case.
func toSnakeCase(str string) string {
	var result []rune
	for i, r := range str {
		if i > 0 && isUpper(r) && (i+1 < len(str) && isLower(rune(str[i+1])) || isLower(rune(str[i-1]))) {
			result = append(result, '_')
		}
		result = append(result, toLowerRune(r))
	}
	return string(result)
}

func isUpper(r rune) bool {
	return 'A' <= r && r <= 'Z'
}

func isLower(r rune) bool {
	return 'a' <= r && r <= 'z'
}

func toLowerRune(r rune) rune {
	if isUpper(r) {
		return r + ('a' - 'A')
	}
	return r
}

// hasError checks if any of the return types is an error
func hasError(returns []Return) bool {
	for _, r := range returns {
		if r.Type == "error" {
			return true
		}
	}
	return false
}

// hasOnlyError checks if the only return type is error
func hasOnlyError(returns []Return) bool {
	if len(returns) != 1 {
		return false
	}
	return returns[0].Type == "error"
}

// hasMultiple checks if there are multiple non-error return types
func hasMultiple(returns []Return) bool {
	count := 0
	for _, r := range returns {
		if r.Type != "error" {
			count++
		}
	}
	return count > 1
}

// Template function to handle optional fields
func orFunc(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
