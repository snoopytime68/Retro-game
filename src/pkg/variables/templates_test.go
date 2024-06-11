// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

package variables

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var start = `
This is a test file for templating.

  ###PREFIX_VAR_REPLACE_ME###
  ###PREFIX_CONST_REPLACE_ME###
  ###PREFIX_APP_REPLACE_ME###
  ###PREFIX_NON_EXIST###
`
var simple = `
This is a test file for templating.

  VAR_REPLACED
  CONST_REPLACED
  APP_REPLACED
  ###PREFIX_NON_EXIST###
`
var multiline = `
This is a test file for templating.

  VAR_REPLACED
VAR_SECOND
  CONST_REPLACED
CONST_SECOND
  APP_REPLACED
APP_SECOND
  ###PREFIX_NON_EXIST###
`
var autoIndent = `
This is a test file for templating.

  VAR_REPLACED
  VAR_SECOND
  CONST_REPLACED
  CONST_SECOND
  APP_REPLACED
  APP_SECOND
  ###PREFIX_NON_EXIST###
`
var file = `
This is a test file for templating.

  The contents of this file become the template value
  CONSTs Don't Support File
  The contents of this file become the template value
  ###PREFIX_NON_EXIST###
`

func TestReplaceTextTemplate(t *testing.T) {
	type test struct {
		vc           VariableConfig
		path         string
		wantErr      bool
		wantContents string
	}

	tests := []test{
		{
			vc:           VariableConfig{setVariableMap: SetVariableMap{}, applicationTemplates: map[string]*TextTemplate{}},
			path:         "non-existent.test",
			wantErr:      true,
			wantContents: start,
		},
		{
			vc: VariableConfig{
				templatePrefix: "PREFIX",
				setVariableMap: SetVariableMap{
					"REPLACE_ME": {Value: "VAR_REPLACED"},
				},
				constants: []Constant{{Name: "REPLACE_ME", Value: "CONST_REPLACED"}},
				applicationTemplates: map[string]*TextTemplate{
					"###PREFIX_APP_REPLACE_ME###": {Value: "APP_REPLACED"},
				},
			},
			wantContents: simple,
		},
		{
			vc: VariableConfig{
				templatePrefix: "PREFIX",
				setVariableMap: SetVariableMap{
					"REPLACE_ME": {Value: "VAR_REPLACED\nVAR_SECOND"},
				},
				constants: []Constant{{Name: "REPLACE_ME", Value: "CONST_REPLACED\nCONST_SECOND"}},
				applicationTemplates: map[string]*TextTemplate{
					"###PREFIX_APP_REPLACE_ME###": {Value: "APP_REPLACED\nAPP_SECOND"},
				},
			},
			wantContents: multiline,
		},
		{
			vc: VariableConfig{
				templatePrefix: "PREFIX",
				setVariableMap: SetVariableMap{
					"REPLACE_ME": {Value: "VAR_REPLACED\nVAR_SECOND", Variable: Variable{AutoIndent: true}},
				},
				constants: []Constant{{Name: "REPLACE_ME", Value: "CONST_REPLACED\nCONST_SECOND", AutoIndent: true}},
				applicationTemplates: map[string]*TextTemplate{
					"###PREFIX_APP_REPLACE_ME###": {Value: "APP_REPLACED\nAPP_SECOND", AutoIndent: true},
				},
			},
			wantContents: autoIndent,
		},
		{
			vc: VariableConfig{
				templatePrefix: "PREFIX",
				setVariableMap: SetVariableMap{
					"REPLACE_ME": {Value: "testdata/file.txt", Variable: Variable{Type: FileVariableType}},
				},
				constants: []Constant{{Name: "REPLACE_ME", Value: "CONSTs Don't Support File"}},
				applicationTemplates: map[string]*TextTemplate{
					"###PREFIX_APP_REPLACE_ME###": {Value: "testdata/file.txt", Type: FileVariableType},
				},
			},
			wantContents: file,
		},
	}

	for _, tc := range tests {
		if tc.path == "" {
			tmpDir := t.TempDir()
			tc.path = filepath.Join(tmpDir, "templates.test")

			f, _ := os.Create(tc.path)
			defer f.Close()

			_, err := f.WriteString(start)
			require.NoError(t, err)
		}

		gotErr := tc.vc.ReplaceTextTemplate(tc.path)
		if tc.wantErr {
			require.Error(t, gotErr)
		} else {
			require.NoError(t, gotErr)
			gotContents, _ := os.ReadFile(tc.path)
			require.Equal(t, tc.wantContents, string(gotContents))
		}
	}
}
