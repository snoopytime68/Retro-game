// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package packager contains functions for linting the zarf.yaml
package packager

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/defenseunicorns/zarf/src/config"
	"github.com/defenseunicorns/zarf/src/pkg/layout"
	"github.com/defenseunicorns/zarf/src/pkg/message"
	"github.com/defenseunicorns/zarf/src/pkg/utils"
	"github.com/defenseunicorns/zarf/src/types"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/xeipuuv/gojsonschema"
)

const (
	zarfInvalidPrefix = "schema is invalid:"
	zarfWarningPrefix = "zarf schema warning:"
	zarfTemplateVar   = "###ZARF_PKG_TMPL_"
)

// ValidateZarfSchema a zarf file against the zarf schema, returns an error if the file is invalid
func (p *Packager) ValidateZarfSchema() (err error) {
	if err = p.readZarfYAML(filepath.Join(p.cfg.CreateOpts.BaseDir, layout.ZarfYAML)); err != nil {
		return fmt.Errorf("unable to read the zarf.yaml file: %s", err.Error())
	}

	if err := checkForVarInComponentImport(p.cfg.Pkg); err != nil {
		message.Warn(err.Error())
	}

	zarfSchema, _ := config.GetSchemaFile()
	var zarfData interface{}
	if err := utils.ReadYaml(filepath.Join(p.cfg.CreateOpts.BaseDir, layout.ZarfYAML), &zarfData); err != nil {
		return err
	}

	if err = validateSchema(zarfData, zarfSchema); err != nil {
		return err
	}

	message.Success("Validation successful")
	return nil
}

func validateSchema2(unmarshalledYaml interface{}, jsonSchema []byte) error {
	schemaLoader := gojsonschema.NewBytesLoader(jsonSchema)
	documentLoader := gojsonschema.NewGoLoader(unmarshalledYaml)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		errorMessage := zarfInvalidPrefix
		for _, desc := range result.Errors() {
			errorMessage = fmt.Sprintf("%s\n - %s", errorMessage, desc.String())
		}
		err = errors.New(errorMessage)
	}

	return err
}

func checkForVarInComponentImport(zarfYaml types.ZarfPackage) error {
	valid := true
	errorMessage := zarfWarningPrefix
	componentWarningStart := "component/"
	for i, component := range zarfYaml.Components {
		if strings.Contains(component.Import.Path, zarfTemplateVar) {
			errorMessage = fmt.Sprintf("%s %s%d/import/path will not resolve ZARF_PKG_TMPL_* variables.",
				errorMessage, componentWarningStart, i)
			valid = false
		}
		if strings.Contains(component.Import.URL, zarfTemplateVar) {
			errorMessage = fmt.Sprintf("%s %s%d/import/url will not resolve ZARF_PKG_TMPL_* variables.",
				errorMessage, componentWarningStart, i)
			valid = false
		}
	}
	if valid {
		return nil
	}
	return errors.New(errorMessage)
}

func validateSchema(unmarshalledYaml interface{}, jsonSchema []byte) error {
	compiler := jsonschema.NewCompiler()
	inMemoryZarfSchema := "schema.json"

	if err := compiler.AddResource(inMemoryZarfSchema, bytes.NewReader(jsonSchema)); err != nil {
		return err
	}

	schema, err := compiler.Compile(inMemoryZarfSchema)
	if err != nil {
		return err
	}

	if err := schema.Validate(unmarshalledYaml); err != nil {
		if validationError, ok := err.(*jsonschema.ValidationError); ok {
			allSchemaErrors := printAllCauses(validationError, []error{})
			var errMessage strings.Builder
			errMessage.WriteString(zarfInvalidPrefix)
			for _, err := range allSchemaErrors {
				errMessage.WriteString("\n")
				errMessage.WriteString(err.Error())
			}
			return errors.New(errMessage.String())
		}
		return err
	}

	return nil
}

func printAllCauses(validationErr *jsonschema.ValidationError, errToUser []error) []error {
	if validationErr == nil {
		return errToUser
	}
	if validationErr.Causes == nil {
		errMessage := fmt.Sprintf(" - %s: %s", validationErr.InstanceLocation, validationErr.Message)
		return append(errToUser, fmt.Errorf("%s", errMessage))
	}

	for _, subCause := range validationErr.Causes {
		errToUser = printAllCauses(subCause, errToUser)
	}
	return errToUser
}
