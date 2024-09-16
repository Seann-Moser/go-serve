package generator

import "github.com/Seann-Moser/go-serve/server/endpoints"

var _ Generator = GoCommentGenerator{}

type GoCommentGenerator struct {
}

func (g GoCommentGenerator) Generate(data GeneratorData, endpoints ...*endpoints.Endpoint) error {

	return nil
}
