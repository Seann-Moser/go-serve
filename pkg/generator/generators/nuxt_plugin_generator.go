package generators

import (
	_ "embed"
	"github.com/Seann-Moser/go-serve/server/endpoints"
)

var _ Generator = NuxtPluginGenerator{}

type NuxtPluginGenerator struct {
}

//go:embed templates/js_function_template.tmpl
var JSFunctionTemplate string

func (n NuxtPluginGenerator) Generate(data GeneratorData, endpoints ...*endpoints.Endpoint) error {
	groupedEndpoints := groupEndpointsByGroup(endpoints) // Group by group name
	_, privateDir, err := GetPublicPrivateDir(data)
	if err != nil {
		return err
	}
	var output []string
	for _, eList := range groupedEndpoints {
		for _, e := range eList {
			for _, cf := range JSNewClientFunc(data.ProjectName, e) {
				c, _ := templ(cf, JSFunctionTemplate)
				output = append(output, c)
			}
		}

	}
	if err := writeNuxtFile(privateDir, data.ProjectName, output, false); err != nil {
		return err
	}

	return nil
}
