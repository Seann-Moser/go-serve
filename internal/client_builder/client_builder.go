package client_builder

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("client-builder", pflag.ExitOnError)
	return fs
}

func Runner(cmd *cobra.Command, args []string) error {
	currentPath, err := os.Getwd()
	if err != nil {
		return err
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	homeDir = path.Join(homeDir, "go", "src") + "/"
	_, projectName := path.Split(currentPath)
	clientDir := fmt.Sprintf("pkg/%s_client", ToSnakeCase(projectName))

	importPath := strings.ReplaceAll(currentPath, homeDir, "")
	ctxLogger.Info(cmd.Context(), "path", zap.String("current_path", currentPath), zap.String("go_import", importPath), zap.String("project_name", projectName), zap.String("dir", clientDir))
	err = createDir(clientDir)
	if err != nil {
		return err
	}

	return nil
}

func createDir(dir string) error {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func ToSnakeCase(str string) string {
	str = strings.ReplaceAll(str, "-", "")
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
