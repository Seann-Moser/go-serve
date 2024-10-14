package generators

import (
	_ "embed"
	"github.com/Seann-Moser/go-serve/pkg/clientpkg"
	"github.com/Seann-Moser/go-serve/server/endpoints"
)

var _ Generator = NuxtPluginGenerator{}

type NuxtPluginGenerator struct {
}

//go:embed templates/js_function_template.tmpl
var JSFunctionTemplate string

//go:embed templates/js_classes.tmpl
var jsClassesTemplate string

func (n NuxtPluginGenerator) Generate(data GeneratorData, endpoints ...*endpoints.Endpoint) error {
	groupedEndpoints := groupEndpointsByGroup(endpoints) // Group by group name
	public, privateDir, err := GetPublicPrivateDir(data)
	if err != nil {
		return err
	}
	var output []string
	var publicOutput []string
	var objects map[string][]string
	var publicObjects map[string][]string
	for _, eList := range groupedEndpoints {
		for _, e := range eList {
			for _, cf := range JSNewClientFunc(data.ProjectName, e) {
				c, _ := templ(cf, JSFunctionTemplate)
				objects = clientpkg.MergeMap[[]string](cf.Objects, objects)
				output = append(output, c)
				if e.Public {
					publicObjects = clientpkg.MergeMap[[]string](cf.Objects, publicObjects)
					publicOutput = append(publicOutput, c)

				}
			}
		}

	}

	if err := writeNuxtFile(privateDir, data.ProjectName, output, false, objects); err != nil {
		return err
	}
	classes, err := templ(objects, jsClassesTemplate)
	if err := writeClassFile(privateDir, data.ProjectName, classes, false); err != nil {
		return err
	}

	if err := writeNuxtFile(public, data.ProjectName, output, true, objects); err != nil {
		return err
	}
	publicClasses, err := templ(objects, jsClassesTemplate)
	if err := writeClassFile(public, data.ProjectName, publicClasses, true); err != nil {
		return err
	}
	return nil
}
