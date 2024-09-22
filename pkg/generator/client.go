package generator

import (
	"fmt"
	generators "github.com/Seann-Moser/go-serve/pkg/generator/generators"
	"github.com/Seann-Moser/go-serve/server/endpoints"
)

type Client struct {
}

func New() *Client {
	return &Client{}
}

func (c *Client) Generate(genList []generators.Generator, endpoints ...*endpoints.Endpoint) error {
	rootDir, err := generators.GetRootDir()
	if err != nil {
		return err
	}
	projectName, err := generators.GetProjectName()
	if err != nil {
		return err
	}
	for _, g := range genList {
		err = g.Generate(generators.GeneratorData{
			ProjectName: projectName,
			RootDir:     rootDir,
		}, endpoints...)
		if err != nil {
			return fmt.Errorf("failed generating data: %w", err)
		}

	}

	return nil
}
