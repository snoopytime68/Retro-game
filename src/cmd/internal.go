// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package cmd contains the CLI commands for Zarf.
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/defenseunicorns/pkg/helpers/v2"
	"github.com/invopop/jsonschema"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/pflag"
	"github.com/zarf-dev/zarf/src/api/v1alpha1"
	"github.com/zarf-dev/zarf/src/cmd/common"
	"github.com/zarf-dev/zarf/src/config/lang"
	"github.com/zarf-dev/zarf/src/internal/agent"
	"github.com/zarf-dev/zarf/src/internal/gitea"
	"github.com/zarf-dev/zarf/src/pkg/cluster"
	"github.com/zarf-dev/zarf/src/pkg/message"
	"github.com/zarf-dev/zarf/src/types"
)

var (
	rollback bool
)

var internalCmd = &cobra.Command{
	Use:    "internal",
	Hidden: true,
	Short:  lang.CmdInternalShort,
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: lang.CmdInternalAgentShort,
	Long:  lang.CmdInternalAgentLong,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cluster, err := cluster.NewCluster()
		if err != nil {
			return err
		}
		return agent.StartWebhook(cmd.Context(), cluster)
	},
}

var httpProxyCmd = &cobra.Command{
	Use:   "http-proxy",
	Short: lang.CmdInternalProxyShort,
	Long:  lang.CmdInternalProxyLong,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cluster, err := cluster.NewCluster()
		if err != nil {
			return err
		}
		return agent.StartHTTPProxy(cmd.Context(), cluster)
	},
}

var genCLIDocs = &cobra.Command{
	Use:   "gen-cli-docs",
	Short: lang.CmdInternalGenerateCliDocsShort,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Don't include the datestamp in the output
		rootCmd.DisableAutoGenTag = true

		resetStringFlags := func(cmd *cobra.Command) {
			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				if flag.Value.Type() == "string" {
					flag.DefValue = ""
				}
			})
		}

		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == "tools" {
				for _, toolCmd := range cmd.Commands() {
					// If the command is a vendored command, add a dummy flag to hide root flags from the docs
					if common.CheckVendorOnlyFromPath(toolCmd) {
						addHiddenDummyFlag(toolCmd, "log-level")
						addHiddenDummyFlag(toolCmd, "architecture")
						addHiddenDummyFlag(toolCmd, "no-log-file")
						addHiddenDummyFlag(toolCmd, "no-progress")
						addHiddenDummyFlag(toolCmd, "zarf-cache")
						addHiddenDummyFlag(toolCmd, "tmpdir")
						addHiddenDummyFlag(toolCmd, "insecure")
						addHiddenDummyFlag(toolCmd, "no-color")
					}

					// Remove the default values from all of the helm commands during the CLI command doc generation
					if toolCmd.Use == "helm" || toolCmd.Use == "sbom" {
						toolCmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
							if flag.Value.Type() == "string" {
								flag.DefValue = ""
							}
						})
						resetStringFlags(toolCmd)
						for _, subCmd := range toolCmd.Commands() {
							resetStringFlags(subCmd)
							for _, helmSubCmd := range subCmd.Commands() {
								resetStringFlags(helmSubCmd)
							}
						}
					}

					if toolCmd.Use == "monitor" {
						resetStringFlags(toolCmd)
					}

					if toolCmd.Use == "yq" {
						for _, subCmd := range toolCmd.Commands() {
							if subCmd.Name() == "shell-completion" {
								subCmd.Hidden = true
							}
						}
					}
				}
			}
		}

		if err := os.RemoveAll("./site/src/content/docs/commands"); err != nil {
			return err
		}
		if err := os.Mkdir("./site/src/content/docs/commands", 0775); err != nil {
			return err
		}

		var prependTitle = func(s string) string {
			fmt.Println(s)

			name := filepath.Base(s)

			// strip .md extension
			name = name[:len(name)-3]

			// replace _ with space
			title := strings.Replace(name, "_", " ", -1)

			return fmt.Sprintf(`---
title: %s
description: Zarf CLI command reference for <code>%s</code>.
tableOfContents: false
---

<!-- Page generated by Zarf; DO NOT EDIT -->

`, title, title)
		}

		var linkHandler = func(link string) string {
			return "/commands/" + link[:len(link)-3] + "/"
		}

		if err := doc.GenMarkdownTreeCustom(rootCmd, "./site/src/content/docs/commands", prependTitle, linkHandler); err != nil {
			return err
		}
		message.Success(lang.CmdInternalGenerateCliDocsSuccess)
		return nil
	},
}

func addGoComments(reflector *jsonschema.Reflector) error {
	addCommentErr := errors.New("this command must be called from the root of the Zarf repo")

	typePackagePath := filepath.Join("src", "api", "v1alpha1")
	if err := reflector.AddGoComments("github.com/zarf-dev/zarf", typePackagePath); err != nil {
		return fmt.Errorf("%w: %w", addCommentErr, err)
	}
	varPackagePath := filepath.Join("src", "pkg", "variables")
	if err := reflector.AddGoComments("github.com/zarf-dev/zarf", varPackagePath); err != nil {
		return fmt.Errorf("%w: %w", addCommentErr, err)
	}
	return nil
}

var genConfigSchemaCmd = &cobra.Command{
	Use:     "gen-config-schema",
	Aliases: []string{"gc"},
	Short:   lang.CmdInternalConfigSchemaShort,
	RunE: func(_ *cobra.Command, _ []string) error {
		reflector := jsonschema.Reflector(jsonschema.Reflector{ExpandedStruct: true})
		if err := addGoComments(&reflector); err != nil {
			return err
		}

		schema := reflector.Reflect(&v1alpha1.ZarfPackage{})
		output, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return fmt.Errorf("unable to generate the Zarf config schema: %w", err)
		}
		fmt.Print(string(output) + "\n")
		return nil
	},
}

type zarfTypes struct {
	DeployedPackage types.DeployedPackage
	ZarfPackage     v1alpha1.ZarfPackage
	ZarfState       types.ZarfState
}

var genTypesSchemaCmd = &cobra.Command{
	Use:     "gen-types-schema",
	Aliases: []string{"gt"},
	Short:   lang.CmdInternalTypesSchemaShort,
	RunE: func(_ *cobra.Command, _ []string) error {
		reflector := jsonschema.Reflector(jsonschema.Reflector{ExpandedStruct: true})
		if err := addGoComments(&reflector); err != nil {
			return err
		}

		schema := reflector.Reflect(&zarfTypes{})
		output, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return fmt.Errorf("unable to generate the JSON schema for the Zarf types DeployedPackage, ZarfPackage, and ZarfState: %w", err)
		}
		fmt.Print(string(output) + "\n")
		return nil
	},
}

var createReadOnlyGiteaUser = &cobra.Command{
	Use:   "create-read-only-gitea-user",
	Short: lang.CmdInternalCreateReadOnlyGiteaUserShort,
	Long:  lang.CmdInternalCreateReadOnlyGiteaUserLong,
	RunE: func(cmd *cobra.Command, _ []string) error {
		timeoutCtx, cancel := context.WithTimeout(cmd.Context(), cluster.DefaultTimeout)
		defer cancel()
		c, err := cluster.NewClusterWithWait(timeoutCtx)
		if err != nil {
			return err
		}
		state, err := c.LoadZarfState(cmd.Context())
		if err != nil {
			return err
		}
		tunnel, err := c.NewTunnel(cluster.ZarfNamespaceName, cluster.SvcResource, cluster.ZarfGitServerName, "", 0, cluster.ZarfGitServerPort)
		if err != nil {
			return err
		}
		_, err = tunnel.Connect(cmd.Context())
		if err != nil {
			return err
		}
		defer tunnel.Close()
		tunnelURL := tunnel.HTTPEndpoint()
		giteaClient, err := gitea.NewClient(tunnelURL, state.GitServer.PushUsername, state.GitServer.PushPassword)
		if err != nil {
			return err
		}
		err = tunnel.Wrap(func() error {
			err = giteaClient.CreateReadOnlyUser(cmd.Context(), state.GitServer.PullUsername, state.GitServer.PullPassword)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	},
}

var createPackageRegistryToken = &cobra.Command{
	Use:   "create-artifact-registry-token",
	Short: lang.CmdInternalArtifactRegistryGiteaTokenShort,
	Long:  lang.CmdInternalArtifactRegistryGiteaTokenLong,
	RunE: func(cmd *cobra.Command, _ []string) error {
		timeoutCtx, cancel := context.WithTimeout(cmd.Context(), cluster.DefaultTimeout)
		defer cancel()
		c, err := cluster.NewClusterWithWait(timeoutCtx)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		state, err := c.LoadZarfState(ctx)
		if err != nil {
			return err
		}

		// If we are setup to use an internal artifact server, create the artifact registry token
		if state.ArtifactServer.IsInternal() {
			tunnel, err := c.NewTunnel(cluster.ZarfNamespaceName, cluster.SvcResource, cluster.ZarfGitServerName, "", 0, cluster.ZarfGitServerPort)
			if err != nil {
				return err
			}
			_, err = tunnel.Connect(cmd.Context())
			if err != nil {
				return err
			}
			defer tunnel.Close()
			tunnelURL := tunnel.HTTPEndpoint()
			giteaClient, err := gitea.NewClient(tunnelURL, state.GitServer.PushUsername, state.GitServer.PushPassword)
			if err != nil {
				return err
			}
			err = tunnel.Wrap(func() error {
				tokenSha1, err := giteaClient.CreatePackageRegistryToken(ctx)
				if err != nil {
					return fmt.Errorf("unable to create an artifact registry token for Gitea: %w", err)
				}
				state.ArtifactServer.PushToken = tokenSha1
				return nil
			})
			if err != nil {
				return err
			}
			if err := c.SaveZarfState(ctx, state); err != nil {
				return err
			}
		}
		return nil
	},
}

var updateGiteaPVC = &cobra.Command{
	Use:   "update-gitea-pvc",
	Short: lang.CmdInternalUpdateGiteaPVCShort,
	Long:  lang.CmdInternalUpdateGiteaPVCLong,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		pvcName := os.Getenv("ZARF_VAR_GIT_SERVER_EXISTING_PVC")

		c, err := cluster.NewCluster()
		if err != nil {
			return err
		}
		// There is a possibility that the pvc does not yet exist and Gitea helm chart should create it
		helmShouldCreate, err := c.UpdateGiteaPVC(ctx, pvcName, rollback)
		if err != nil {
			message.WarnErr(err, lang.CmdInternalUpdateGiteaPVCErr)
		}
		fmt.Print(helmShouldCreate)
		return nil
	},
}

var isValidHostname = &cobra.Command{
	Use:   "is-valid-hostname",
	Short: lang.CmdInternalIsValidHostnameShort,
	RunE: func(_ *cobra.Command, _ []string) error {
		if valid := helpers.IsValidHostName(); !valid {
			hostname, err := os.Hostname()
			return fmt.Errorf("the hostname %s is not valid. Ensure the hostname meets RFC1123 requirements https://www.rfc-editor.org/rfc/rfc1123.html, error=%w", hostname, err)
		}
		return nil
	},
}

var computeCrc32 = &cobra.Command{
	Use:     "crc32 TEXT",
	Aliases: []string{"c"},
	Short:   lang.CmdInternalCrc32Short,
	Args:    cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		text := args[0]
		hash := helpers.GetCRCHash(text)
		fmt.Printf("%d\n", hash)
	},
}

func init() {
	rootCmd.AddCommand(internalCmd)

	internalCmd.AddCommand(agentCmd)
	internalCmd.AddCommand(httpProxyCmd)
	internalCmd.AddCommand(genCLIDocs)
	internalCmd.AddCommand(genConfigSchemaCmd)
	internalCmd.AddCommand(genTypesSchemaCmd)
	internalCmd.AddCommand(createReadOnlyGiteaUser)
	internalCmd.AddCommand(createPackageRegistryToken)
	internalCmd.AddCommand(updateGiteaPVC)
	internalCmd.AddCommand(isValidHostname)
	internalCmd.AddCommand(computeCrc32)

	updateGiteaPVC.Flags().BoolVarP(&rollback, "rollback", "r", false, lang.CmdInternalFlagUpdateGiteaPVCRollback)
}

func addHiddenDummyFlag(cmd *cobra.Command, flagDummy string) {
	if cmd.PersistentFlags().Lookup(flagDummy) == nil {
		var dummyStr string
		cmd.PersistentFlags().StringVar(&dummyStr, flagDummy, "", "")
		err := cmd.PersistentFlags().MarkHidden(flagDummy)
		if err != nil {
			slog.Debug("Unable to add hidden dummy flag", "error", err)
		}
	}
}
