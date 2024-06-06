package generator

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
)

func GetRootDir() (string, error) {
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
	return rootDir, nil
}

func GetProjectName() (string, error) {
	rootDir, err := GetRootDir()
	if err != nil {
		return "", fmt.Errorf("failed to get project name: %v", err)
	}
	_, projectName := path.Split(rootDir)
	return projectName, nil
}
