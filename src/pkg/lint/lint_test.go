// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package lint contains functions for verifying zarf yaml files are valid
package lint

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zarf-dev/zarf/src/api/v1alpha1"
	"github.com/zarf-dev/zarf/src/config/lang"
)

func TestLintError(t *testing.T) {
	t.Parallel()

	lintErr := &LintError{
		Findings: []PackageFinding{
			{
				Severity: SevWarn,
			},
		},
	}
	require.Equal(t, "linting error found 1 instance(s)", lintErr.Error())
	require.True(t, lintErr.OnlyWarnings())

	lintErr = &LintError{
		Findings: []PackageFinding{
			{
				Severity: SevWarn,
			},
			{
				Severity: SevErr,
			},
		},
	}
	require.Equal(t, "linting error found 2 instance(s)", lintErr.Error())
	require.False(t, lintErr.OnlyWarnings())
}

func TestLintComponents(t *testing.T) {
	t.Run("Test composable components with bad path", func(t *testing.T) {
		t.Parallel()
		zarfPackage := v1alpha1.ZarfPackage{
			Components: []v1alpha1.ZarfComponent{
				{
					Import: v1alpha1.ZarfComponentImport{Path: "bad-path"},
				},
			},
			Metadata: v1alpha1.ZarfMetadata{Name: "test-zarf-package"},
		}

		_, err := lintComponents(context.Background(), zarfPackage, "", nil)
		require.Error(t, err)
	})
}
func TestFillObjTemplate(t *testing.T) {
	SetVariables := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}

	component := v1alpha1.ZarfComponent{
		Images: []string{
			fmt.Sprintf("%s%s###", v1alpha1.ZarfPackageTemplatePrefix, "KEY1"),
			fmt.Sprintf("%s%s###", v1alpha1.ZarfPackageVariablePrefix, "KEY2"),
			fmt.Sprintf("%s%s###", v1alpha1.ZarfPackageTemplatePrefix, "KEY3"),
		},
	}

	findings, err := templateZarfObj(&component, SetVariables)
	require.NoError(t, err)
	expectedFindings := []PackageFinding{
		{
			Severity:    SevWarn,
			Description: "There are templates that are not set and won't be evaluated during lint",
		},
		{
			Severity:    SevWarn,
			Description: fmt.Sprintf(lang.PkgValidateTemplateDeprecation, "KEY2", "KEY2", "KEY2"),
		},
	}
	expectedComponent := v1alpha1.ZarfComponent{
		Images: []string{
			"value1",
			"value2",
			fmt.Sprintf("%s%s###", v1alpha1.ZarfPackageTemplatePrefix, "KEY3"),
		},
	}
	require.ElementsMatch(t, expectedFindings, findings)
	require.Equal(t, expectedComponent, component)
}
