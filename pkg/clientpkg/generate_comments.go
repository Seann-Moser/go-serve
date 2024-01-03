package clientpkg

import (
	"bufio"
	_ "embed"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/response"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// https://github.com/swaggo/swag
type ApiDoc struct {
	Title       string
	Version     string
	Description string
	Host        string
}

type SwagEndpoint struct {
	FuncName string
	Summary  string
	Tags     string
	ID       string
	Produce  string // mime types https://github.com/swaggo/swag#mime-types
	Path     string
	Methods  []string

	Params    []SwagParams
	Successes []ReturnStatus
	Failures  []ReturnStatus
}
type SwagParams struct {
	Name        string
	Location    string // query/path/header/body/formData https://github.com/swaggo/swag#data-type
	Type        string // string/booleal/int/number/file/any struct
	Required    bool
	Description string
}

type ReturnStatus struct {
	Status    int
	ParamType string //object/array
	DataType  string
	Message   string
}

//go:embed templates/swag_api.tmpl
var SwagApiGeneralTempl string

//go:embed templates/swag_endpoint.tmpl
var SwagApiEndpointTempl string

func GenerateComments(doc *ApiDoc, endpoints ...*endpoints.Endpoint) {
	projectPath, _, err := GetProjectDir()
	if err != nil {
		return
	}
	goFiles := GetGoFiles(projectPath)
	functions := map[string]Func{} //todo fix to allow for multiple funcs in same file
	for _, e := range endpoints {
		fullName := GetFunctionName(e.HandlerFunc)
		if fullName == "" {
			continue
		}
		fullName = strings.TrimSuffix(fullName, "-fm")
		_, pkg := path.Split(fullName)
		tmpPkg := strings.Split(pkg, ".")
		pkg = strings.Join(tmpPkg[1:], " ")
		pkg = strings.TrimPrefix(pkg, "(*")
		pkg = strings.ReplaceAll(pkg, ")", `\)`)
		tmp := FindFunction(pkg, goFiles)
		for _, t := range tmp {
			t.FormatComment(e)
			_ = t.UpdateComment()
		}
		functions = MergeMap[Func](functions, tmp)
	}
	api, err := templateReplace(SwagApiGeneralTempl, doc)
	if err != nil {
		return
	}

	data := FindFunction("main", goFiles)
	if len(data) == 0 {
		return
	}
	for _, v := range data {
		v.Comment.Set(api)
		_ = v.UpdateComment()
		break
	}

}

func GetFullName(i interface{}) string {
	fullPkg := getTypePkg(i)
	_, pkg := path.Split(fullPkg)
	if isMap(i) {
		return fmt.Sprintf("map[%s]%s", getType(i), getType(i))
	} else if isArray(i) {
		if getType(i) == "BaseResponse" {
			return fmt.Sprintf("response.BaseResponse")
		}
		return fmt.Sprintf("response.BaseResponse{data=[]%s.%s}", pkg, getType(i))
	} else {
		if getType(i) == "BaseResponse" {
			return fmt.Sprintf("response.BaseResponse")
		}

		return fmt.Sprintf("response.BaseResponse{data=%s.%s}", pkg, getType(i))
	}
}

func FindFunction(fName string, goFiles []string) map[string]Func {
	found := map[string]Func{}
	for _, files := range goFiles {
		if cmt, ln, err := FindString(files, regexp.MustCompile(`func[\(\s\*a-z]*`+fName+`\s{0,1}\(`)); err == nil && ln > 0 {
			found[files] = Func{
				File:    files,
				Name:    fName,
				Ln:      ln,
				Comment: cmt,
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

type Func struct {
	File    string
	Name    string
	Ln      int
	Comment *Comment
}

func (fc *Func) FormatComment(endpoint *endpoints.Endpoint) {
	params := []SwagParams{}
	successes := []ReturnStatus{}
	failures := []ReturnStatus{}
	re := regexp.MustCompile(`\{(.*?)\}`)
	muxVars := re.FindAllString(endpoint.URLPath, -1)
	for i := range muxVars {
		n := regexp.MustCompile(`[\{\}]`).ReplaceAllString(muxVars[i], "")
		muxVars[i] = n
		params = append(params, SwagParams{
			Name:        n,
			Location:    "path",
			Type:        "string",
			Required:    true,
			Description: "todo",
		})
	}
	for _, p := range endpoint.QueryParams {
		params = append(params, SwagParams{
			Name:        p,
			Location:    "query",
			Type:        "string",
			Required:    false,
			Description: "todo",
		})
	}

	for _, p := range endpoint.Headers {
		params = append(params, SwagParams{
			Name:        p,
			Location:    "header",
			Type:        "string",
			Required:    false,
			Description: "todo",
		})
	}

	for _, v := range endpoint.RequestTypeMap {
		n := snakeCaseToCamelCase(ToSnakeCase(getType(v)))
		if len(n) == 0 {
			log.Fatal("unable to get name for requesttype")
		}
		if isMap(v) {
			n += "Map"
		}
		fullPkg := getTypePkg(v)
		_, pkg := path.Split(fullPkg)
		n = strings.ToLower(n[:1]) + n[1:]
		t := fmt.Sprintf("%s.%s", pkg, getType(v))
		if pkg == "" {
			pkg = t
		}
		params = append(params, SwagParams{
			Name:        n,
			Location:    "body",
			Type:        t,
			Required:    false,
			Description: "todo",
		})
	}

	for _, v := range endpoint.ResponseTypeMap {
		paramType := "object"
		successes = append(successes, ReturnStatus{
			Status:    http.StatusOK,
			ParamType: paramType,
			DataType:  GetFullName(v),
			Message:   "todo",
		})

	}
	if len(successes) == 0 {
		successes = append(successes, ReturnStatus{
			Status:    http.StatusOK,
			ParamType: "object",
			DataType:  GetFullName(response.BaseResponse{}),
			Message:   "todo",
		})
	}

	failures = append(failures, ReturnStatus{
		Status:    http.StatusBadRequest,
		ParamType: "object",
		DataType:  GetFullName(response.BaseResponse{}),
		Message:   "todo",
	})
	failures = append(failures, ReturnStatus{
		Status:    http.StatusInternalServerError,
		ParamType: "object",
		DataType:  GetFullName(response.BaseResponse{}),
		Message:   "todo",
	})
	failures = append(failures, ReturnStatus{
		Status:    http.StatusUnauthorized,
		ParamType: "object",
		DataType:  GetFullName(response.BaseResponse{}),
		Message:   "todo",
	})
	fullName := GetFunctionName(endpoint.HandlerFunc)
	fullName = strings.TrimSuffix(fullName, "-fm")
	_, pkg := path.Split(fullName)
	tmpPkg := strings.Split(pkg, ".")

	name := SwagEndpoint{
		FuncName:  tmpPkg[len(tmpPkg)-1],
		Summary:   "todo",
		Tags:      path.Base(endpoint.URLPath),
		ID:        ToSnakeCase(UrlToName(endpoint.URLPath)) + "-" + strings.Join(endpoint.Methods, "-"),
		Produce:   "json", //todo or image
		Path:      endpoint.URLPath,
		Methods:   endpoint.Methods,
		Params:    params,
		Successes: successes,
		Failures:  failures,
	}
	cmt, err := templateReplace(SwagApiEndpointTempl, name)
	if err == nil {
		fc.Comment.Set(cmt)
	}
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

func FindString(file string, find *regexp.Regexp) (*Comment, int, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, 0, err
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
	for scanner.Scan() {
		text := scanner.Text()
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
			return comment, line, nil
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
		// Handle the error
	}
	return nil, 0, err
}
func GetGoFiles(path string) []string {
	libRegEx, e := regexp.Compile("^.+\\.(go)$")
	if e != nil {
		log.Fatal(e)
	}
	var files []string
	e = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
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
