package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/invopop/jsonschema"
	"github.com/zarf-dev/zarf/src/api/v1alpha1"
	"github.com/zarf-dev/zarf/src/api/v1beta1"
)

func addGoComments(reflector *jsonschema.Reflector, apiVersion string) error {
	// Get the file path of the currently executing file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("error getting file path")
	}

	typePackagePath := filepath.Join(filename, "..", "..", "..", "src", "api", apiVersion)
	if err := reflector.AddGoComments("github.com/zarf-dev/zarf", typePackagePath); err != nil {
		return err
	}

	return nil
}

var apiVersionToObject = map[string]interface{}{
	"v1alpha1": &v1alpha1.ZarfPackage{},
	"v1beta1":  &v1beta1.ZarfPackage{},
}

func genSchema(apiVersion string) (string, error) {
	reflector := jsonschema.Reflector(jsonschema.Reflector{ExpandedStruct: true})
	if err := addGoComments(&reflector, apiVersion); err != nil {
		return "", err
	}

	schema := reflector.Reflect(apiVersionToObject[apiVersion])
	output, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("unable to generate the Zarf config schema: %w", err)
	}
	return string(output), nil
}

func main() {
	var apiVersions = []string{"v1alpha1", "v1beta1"}
	if len := len(os.Args); len != 2 {
		fmt.Println("This program must be called with the apiVersion, options are", apiVersions)
		os.Exit(1)
	}
	schema, err := genSchema(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(schema)
}
