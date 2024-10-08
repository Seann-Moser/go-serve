package generator

import (
	"bytes"
	"fmt"
	generators "github.com/Seann-Moser/go-serve/pkg/generator/generators"
	"github.com/Seann-Moser/go-serve/server/endpoints"
	"os/exec"
	"strings"
)

type Client struct {
}

func New() *Client {
	return &Client{}
}

func (c *Client) Generate(genList []generators.Generator, host string, description string, endpoints ...*endpoints.Endpoint) error {
	rootDir, err := generators.GetRootDir()
	if err != nil {
		return err
	}
	projectName, err := generators.GetProjectName()
	if err != nil {
		return err
	}
	version, err := GetLatestGitTag()
	if err != nil {
		return err
	}
	for _, g := range genList {
		err = g.Generate(generators.GeneratorData{
			ProjectName: projectName,
			RootDir:     rootDir,
			Title:       projectName,
			Version:     version,
			Description: description,
			Host:        host,
		}, endpoints...)
		if err != nil {
			return fmt.Errorf("failed generating data: %w", err)
		}

	}

	return nil
}

func GetLatestGitTag() (string, error) {
	// Prepare the git command to get the latest tag
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get latest git tag: %v, %s", err, stderr.String())
	}

	// Return the latest tag, removing any extra whitespace or newlines
	return strings.TrimSpace(out.String()), nil
}
