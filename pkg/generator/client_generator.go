package generator

import "github.com/Seann-Moser/go-serve/server/endpoints"

var _ Generator = GoClientGenerator{}

type GoClientGenerator struct {
}

func (g GoClientGenerator) Generate(data GeneratorData, endpoint *endpoints.Endpoint) error {

	return nil
}
