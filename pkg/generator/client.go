package generator

import (
	"fmt"
	"github.com/Seann-Moser/go-serve/server/endpoints"
)

type GeneratorData struct {
	ProjectName string
	RootDir     string
}

type Generator interface {
	Generate(data GeneratorData, endpoints *endpoints.Endpoint) error
}

type Client struct {
	write     bool
	directory string
}

func New() *Client {
	return &Client{}
}

func (c *Client) Generate(generators []Generator, endpoints ...*endpoints.Endpoint) error {
	rootDir, err := GetRootDir()
	if err != nil {
		return err
	}
	projectName, err := GetProjectName()
	if err != nil {
		return err
	}
	for _, e := range endpoints {
		for _, g := range generators {
			err = g.Generate(GeneratorData{
				ProjectName: projectName,
				RootDir:     rootDir,
			}, e)
			if err != nil {
				return fmt.Errorf("failed generating data (%s): %w", e.URLPath, err)
			}
		}
	}

	return nil
}
