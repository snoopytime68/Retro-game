// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package cmd contains the CLI commands for Zarf.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/jsonschema"
	"github.com/defenseunicorns/zarf/src/config/lang"
	"github.com/defenseunicorns/zarf/src/internal/agent"
	"github.com/defenseunicorns/zarf/src/internal/api"
	"github.com/defenseunicorns/zarf/src/internal/cluster"
	"github.com/defenseunicorns/zarf/src/internal/packager/git"
	"github.com/defenseunicorns/zarf/src/pkg/message"
	"github.com/defenseunicorns/zarf/src/pkg/utils"
	"github.com/defenseunicorns/zarf/src/types"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var internalCmd = &cobra.Command{
	Use:     "internal",
	Aliases: []string{"dev"},
	Hidden:  true,
	Short:   lang.CmdInternalShort,
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: lang.CmdInternalAgentShort,
	Long:  lang.CmdInternalAgentLong,
	Run: func(cmd *cobra.Command, args []string) {
		agent.StartWebhook()
	},
}

var httpProxyCmd = &cobra.Command{
	Use:   "http-proxy",
	Short: lang.CmdInternalProxyShort,
	Long:  lang.CmdInternalProxyLong,
	Run: func(cmd *cobra.Command, args []string) {
		agent.StartHTTPProxy()
	},
}

var generateCLIDocs = &cobra.Command{
	Use:   "generate-cli-docs",
	Short: lang.CmdInternalGenerateCliDocsShort,
	Run: func(cmd *cobra.Command, args []string) {
		// Don't include the datestamp in the output
		rootCmd.DisableAutoGenTag = true
		//Generate markdown of the Zarf command (and all of its child commands)
		if err := os.RemoveAll("./docs/2-the-zarf-cli/100-cli-commands"); err != nil {
			message.Fatalf(lang.CmdInternalGenerateCliDocsErr, err.Error())
		}
		if err := os.Mkdir("./docs/2-the-zarf-cli/100-cli-commands", 0775); err != nil {
			message.Fatalf(lang.CmdInternalGenerateCliDocsErr, err.Error())
		}
		if err := doc.GenMarkdownTree(rootCmd, "./docs/2-the-zarf-cli/100-cli-commands"); err != nil {
			message.Fatalf(lang.CmdInternalGenerateCliDocsErr, err.Error())
		} else {
			message.Success(lang.CmdInternalGenerateCliDocsSuccess)
		}
	},
}

var configSchemaCmd = &cobra.Command{
	Use:     "config-schema",
	Aliases: []string{"c"},
	Short:   lang.CmdInternalConfigSchemaShort,
	Run: func(cmd *cobra.Command, args []string) {
		schema := jsonschema.Reflect(&types.ZarfPackage{})
		output, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			message.Fatal(err, lang.CmdInternalConfigSchemaErr)
		}
		fmt.Print(string(output) + "\n")
	},
}

var apiSchemaCmd = &cobra.Command{
	Use:   "api-schema",
	Short: lang.CmdInternalAPISchemaShort,
	Run: func(cmd *cobra.Command, args []string) {
		schema := jsonschema.Reflect(&types.RestAPI{})
		output, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			message.Fatal(err, lang.CmdInternalAPISchemaGenerateErr)
		}
		fmt.Print(string(output) + "\n")
	},
}

var createReadOnlyGiteaUser = &cobra.Command{
	Use:   "create-read-only-gitea-user",
	Short: lang.CmdInternalCreateReadOnlyGiteaUserShort,
	Long:  lang.CmdInternalCreateReadOnlyGiteaUserLong,
	Run: func(cmd *cobra.Command, args []string) {
		// Load the state so we can get the credentials for the admin git user
		state, err := cluster.NewClusterOrDie().LoadZarfState()
		if err != nil {
			message.WarnErr(err, lang.ErrLoadState)
		}

		// Create the non-admin user
		if err = git.New(state.GitServer).CreateReadOnlyUser(); err != nil {
			message.WarnErr(err, lang.CmdInternalCreateReadOnlyGiteaUserErr)
		}
	},
}

var createPackageRegistryToken = &cobra.Command{
	Use:   "create-artifact-registry-token",
	Short: lang.CmdInternalArtifactRegistryGiteaTokenShort,
	Long:  lang.CmdInternalArtifactRegistryGiteaTokenLong,
	Run: func(cmd *cobra.Command, args []string) {
		// Load the state so we can get the credentials for the admin git user
		c := cluster.NewClusterOrDie()
		state, err := c.LoadZarfState()
		if err != nil {
			message.WarnErr(err, lang.ErrLoadState)
		}

		// If we are setup to use an internal artifact server, create the artifact registry token
		if state.ArtifactServer.InternalServer {
			token, err := git.New(state.GitServer).CreatePackageRegistryToken()
			if err != nil {
				message.WarnErr(err, lang.CmdInternalArtifactRegistryGiteaTokenErr)
			}

			state.ArtifactServer.PushToken = token.Sha1

			c.SaveZarfState(state)
		}
	},
}

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: lang.CmdInternalUIShort,
	Long:  lang.CmdInternalUILong,
	Run: func(cmd *cobra.Command, args []string) {
		api.LaunchAPIServer()
	},
}

var isValidHostname = &cobra.Command{
	Use:   "is-valid-hostname",
	Short: lang.CmdInternalIsValidHostnameShort,
	Run: func(cmd *cobra.Command, args []string) {
		if valid := utils.IsValidHostName(); !valid {
			hostname, _ := os.Hostname()
			message.Fatalf(nil, lang.CmdInternalIsValidHostnameErr, hostname)
		}
	},
}

func init() {
	rootCmd.AddCommand(internalCmd)

	internalCmd.AddCommand(agentCmd)
	internalCmd.AddCommand(httpProxyCmd)
	internalCmd.AddCommand(generateCLIDocs)
	internalCmd.AddCommand(configSchemaCmd)
	internalCmd.AddCommand(apiSchemaCmd)
	internalCmd.AddCommand(createReadOnlyGiteaUser)
	internalCmd.AddCommand(createPackageRegistryToken)
	internalCmd.AddCommand(uiCmd)
	internalCmd.AddCommand(isValidHostname)
}
