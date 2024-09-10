// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package common handles command configuration across all commands
package common

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/viper"
	"github.com/zarf-dev/zarf/src/config"
	"github.com/zarf-dev/zarf/src/config/lang"
	"github.com/zarf-dev/zarf/src/pkg/message"
)

// Constants for use when loading configurations from viper config files
const (

	// Root config keys

	VLogLevel     = "log_level"
	VArchitecture = "architecture"
	VNoLogFile    = "no_log_file"
	VNoProgress   = "no_progress"
	VNoColor      = "no_color"
	VZarfCache    = "zarf_cache"
	VTmpDir       = "tmp_dir"
	VInsecure     = "insecure"

	// Init config keys

	VInitComponents   = "init.components"
	VInitStorageClass = "init.storage_class"

	// Init Git config keys

	VInitGitURL      = "init.git.url"
	VInitGitPushUser = "init.git.push_username"
	VInitGitPushPass = "init.git.push_password"
	VInitGitPullUser = "init.git.pull_username"
	VInitGitPullPass = "init.git.pull_password"

	// Init Registry config keys

	VInitRegistryURL      = "init.registry.url"
	VInitRegistryNodeport = "init.registry.nodeport"
	VInitRegistrySecret   = "init.registry.secret"
	VInitRegistryPushUser = "init.registry.push_username"
	VInitRegistryPushPass = "init.registry.push_password"
	VInitRegistryPullUser = "init.registry.pull_username"
	VInitRegistryPullPass = "init.registry.pull_password"

	// Init Package config keys

	VInitArtifactURL       = "init.artifact.url"
	VInitArtifactPushUser  = "init.artifact.push_username"
	VInitArtifactPushToken = "init.artifact.push_token"

	// Package config keys

	VPkgOCIConcurrency = "package.oci_concurrency"
	VPkgPublicKey      = "package.public_key"

	// Package create config keys

	VPkgCreateSet                = "package.create.set"
	VPkgCreateOutput             = "package.create.output"
	VPkgCreateSbom               = "package.create.sbom"
	VPkgCreateSbomOutput         = "package.create.sbom_output"
	VPkgCreateSkipSbom           = "package.create.skip_sbom"
	VPkgCreateMaxPackageSize     = "package.create.max_package_size"
	VPkgCreateSigningKey         = "package.create.signing_key"
	VPkgCreateSigningKeyPassword = "package.create.signing_key_password"
	VPkgCreateDifferential       = "package.create.differential"
	VPkgCreateRegistryOverride   = "package.create.registry_override"
	VPkgCreateFlavor             = "package.create.flavor"

	// Package deploy config keys

	VPkgDeploySet          = "package.deploy.set"
	VPkgDeployComponents   = "package.deploy.components"
	VPkgDeployShasum       = "package.deploy.shasum"
	VPkgDeploySget         = "package.deploy.sget"
	VPkgDeploySkipWebhooks = "package.deploy.skip_webhooks"
	VPkgDeployTimeout      = "package.deploy.timeout"
	VPkgRetries            = "package.deploy.retries"

	// Package publish config keys

	VPkgPublishSigningKey         = "package.publish.signing_key"
	VPkgPublishSigningKeyPassword = "package.publish.signing_key_password"

	// Package pull config keys

	VPkgPullOutputDir = "package.pull.output_directory"

	// Dev deploy config keys

	VDevDeployNoYolo = "dev.deploy.no_yolo"
)

var (
	// Viper instance used by commands
	v *viper.Viper

	// Viper configuration error
	vConfigError error
)

// InitViper initializes the viper singleton for the CLI
func InitViper() *viper.Viper {
	// Already initialized by some other command
	if v != nil {
		return v
	}

	v = viper.New()

	// Skip for vendor-only commands or the version command
	if CheckVendorOnlyFromArgs() || isVersionCmd() {
		return v
	}

	// Specify an alternate config file
	cfgFile := os.Getenv("ZARF_CONFIG")

	// Don't forget to read config either from cfgFile or from home directory!
	if cfgFile != "" {
		// Use config file from the flag.
		v.SetConfigFile(cfgFile)
	} else {
		// Search config paths in the current directory and $HOME/.zarf.
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.zarf")
		v.SetConfigName("zarf-config")
	}

	// E.g. ZARF_LOG_LEVEL=debug
	v.SetEnvPrefix("zarf")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Optional, so ignore errors
	vConfigError = v.ReadInConfig()

	// Set default values for viper
	setDefaults()

	return v
}

// GetViper returns the viper singleton
func GetViper() *viper.Viper {
	return v
}

func isVersionCmd() bool {
	args := os.Args
	return len(args) > 1 && (args[1] == "version" || args[1] == "v")
}

func printViperConfigUsed() {
	// Only print config info if viper is initialized.
	vInitialized := v != nil
	if !vInitialized {
		return
	}
	var notFoundErr viper.ConfigFileNotFoundError
	if errors.As(vConfigError, &notFoundErr) {
		return
	}
	if vConfigError != nil {
		message.WarnErrf(vConfigError, lang.CmdViperErrLoadingConfigFile, vConfigError.Error())
		return
	}
	if cfgFile := v.ConfigFileUsed(); cfgFile != "" {
		message.Notef(lang.CmdViperInfoUsingConfigFile, cfgFile)
	}
}

func setDefaults() {
	// Root defaults that are non-zero values
	v.SetDefault(VLogLevel, "info")
	v.SetDefault(VZarfCache, config.ZarfDefaultCachePath)

	// Package defaults that are non-zero values
	v.SetDefault(VPkgOCIConcurrency, 3)
	v.SetDefault(VPkgRetries, config.ZarfDefaultRetries)

	// Deploy opts that are non-zero values
	v.SetDefault(VPkgDeployTimeout, config.ZarfDefaultTimeout)
}
