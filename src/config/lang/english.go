//go:build !alt_language

// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package lang contains the language strings for english used by Zarf
// Alternative languages can be created by duplicating this file and changing the build tag to "//go:build alt_language && <language>"
package lang

import "errors"

// All language strings should be in the form of a constant
// The constants should be grouped by the top level package they are used in (or common)
// The format should be <PathName><Err/Info><ShortDescription>
// Debug messages will not be a part of the language strings since they are not intended to be user facing
// Include sprintf formatting directives in the string if needed
const (
	ErrNoClusterConnection = "Failed to connect to the Kubernetes cluster."
	ErrTunnelFailed        = "Failed to create a tunnel to the Kubernetes cluster."
	ErrLoadState           = "Failed to load the Zarf State from the Kubernetes cluster."
	ErrUnmarshal           = "failed to unmarshal file: %w"
	ErrMarshal             = "failed to marshal file: %w"
	ErrLoadingConfig       = "failed to load config: %w"
	ErrWritingFile         = "failed to write the file %s: %w"
)

// Zarf CLI commands
const (
	// root zarf command
	RootCmdShort = "DevSecOps for Airgap"
	RootCmdLong  = "Zarf eliminates the complexity of air gap software delivery for Kubernetes clusters and cloud native workloads\n" +
		"using a declarative packaging strategy to support DevSecOps in offline and semi-connected environments."

	RootCmdFlagLogLevel    = "Log level when running Zarf. Valid options are: warn, info, debug, trace"
	RootCmdFlagArch        = "Architecture for OCI images"
	RootCmdFlagSkipLogFile = "Disable log file creation"
	RootCmdFlagNoProgress  = "Disable fancy UI progress bars, spinners, logos, etc"
	RootCmdFlagCachePath   = "Specify the location of the Zarf cache directory"
	RootCmdFlagTempDir     = "Specify the temporary directory to use for intermediate files"

	RootCmdDeprecatedDeploy = "Please use \"zarf package deploy %s\" to deploy this package."
	RootCmdDeprecatedCreate = "Please use \"zarf package create\" to create this package."

	RootCmdErrInvalidLogLevel = "Invalid log level. Valid options are: warn, info, debug, trace."

	// zarf connect
	CmdConnectShort = "Access services or pods deployed in the cluster."
	CmdConnectLong  = "Uses a k8s port-forward to connect to resources within the cluster referenced by your kube-context.\n" +
		"Three default options for this command are <REGISTRY|LOGGING|GIT>. These will connect to the Zarf created resources " +
		"(assuming they were selected when performing the `zarf init` command).\n\n" +
		"Packages can provide service manifests that define their own shortcut connection options. These options will be " +
		"printed to the terminal when the package finishes deploying.\n If you don't remember what connection shortcuts your deployed " +
		"package offers, you can search your cluster for services that have the 'zarf.dev/connect-name' label. The value of that label is " +
		"the name you will pass into the 'zarf connect' command. \n\n" +
		"Even if the packages you deploy don't define their own shortcut connection options, you can use the command flags " +
		"to connect into specific resources. You can read the command flag descriptions below to get a better idea how to connect " +
		"to whatever resource you are trying to connect to."

	// zarf connect list
	CmdConnectListShort = "List all available connection shortcuts."

	CmdConnectFlagName       = "Specify the resource name.  E.g. name=unicorns or name=unicorn-pod-7448499f4d-b5bk6"
	CmdConnectFlagNamespace  = "Specify the namespace.  E.g. namespace=default"
	CmdConnectFlagType       = "Specify the resource type.  E.g. type=svc or type=pod"
	CmdConnectFlagLocalPort  = "(Optional, autogenerated if not provided) Specify the local port to bind to.  E.g. local-port=42000"
	CmdConnectFlagRemotePort = "Specify the remote port of the resource to bind to.  E.g. remote-port=8080"
	CmdConnectFlagCliOnly    = "Disable browser auto-open"

	// zarf destroy
	CmdDestroyShort = "Tear it all down, we'll miss you Zarf..."
	CmdDestroyLong  = "Tear down Zarf.\n\n" +
		"Deletes everything in the 'zarf' namespace within your connected k8s cluster.\n\n" +
		"If Zarf deployed your k8s cluster, this command will also tear your cluster down by " +
		"searching through /opt/zarf for any scripts that start with 'zarf-clean-' and executing them. " +
		"Since this is a cleanup operation, Zarf will not stop the teardown if one of the scripts produce " +
		"an error.\n\n" +
		"If Zarf did not deploy your k8s cluster, this command will delete the Zarf namespace, delete secrets " +
		"and labels that only Zarf cares about, and optionally uninstall components that Zarf deployed onto " +
		"the cluster. Since this is a cleanup operation, Zarf will not stop the uninstalls if one of the " +
		"resources produce an error while being deleted."

	CmdDestroyFlagConfirm          = "REQUIRED. Confirm the destroy action to prevent accidental deletions"
	CmdDestroyFlagRemoveComponents = "Also remove any installed components outside the zarf namespace"

	CmdDestroyErrNoScriptPath           = "Unable to find the folder (%s) which has the scripts to cleanup the cluster. Please double-check you have the right kube-context"
	CmdDestroyErrScriptPermissionDenied = "Received 'permission denied' when trying to execute the script (%s). Please double-check you have the correct kube-context."

	// zarf init
	CmdInitShort = "Prepares a k8s cluster for the deployment of Zarf packages"
	CmdInitLong  = "Injects a docker registry as well as other optional useful things (such as a git server " +
		"and a logging stack) into a k8s cluster under the 'zarf' namespace " +
		"to support future application deployments. \n" +

		"If you do not have a k8s cluster already configured, this command will give you " +
		"the ability to install a cluster locally.\n\n" +

		"This command looks for a zarf-init package in the local directory that the command was executed " +
		"from. If no package is found in the local directory and the Zarf CLI exists somewhere outside of " +
		"the current directory, Zarf will failover and attempt to find a zarf-init package in the directory " +
		"that the Zarf binary is located in.\n\n\n\n" +

		"Example Usage:\n" +
		"# Initializing without any optional components:\nzarf init\n\n" +
		"# Initializing w/ Zarfs internal git server:\nzarf init --components=git-server\n\n" +
		"# Initializing w/ Zarfs internal git server and PLG stack:\nzarf init --components=git-server,logging\n\n" +
		"# Initializing w/ an internal registry but with a different nodeport:\nzarf init --nodeport=30333\n\n" +
		"# Initializing w/ an external registry:\nzarf init --registry-push-password={PASSWORD} --registry-push-username={USERNAME} --registry-url={URL}\n\n" +
		"# Initializing w/ an external git server:\nzarf init --git-push-password={PASSWORD} --git-push-username={USERNAME} --git-url={URL}\n\n"

	CmdInitErrFlags            = "Invalid command flags were provided."
	CmdInitErrDownload         = "failed to download the init package: %w"
	CmdInitErrValidateGit      = "the 'git-push-username' and 'git-push-password' flags must be provided if the 'git-url' flag is provided"
	CmdInitErrValidateRegistry = "the 'registry-push-username' and 'registry-push-password' flags must be provided if the 'registry-url' flag is provided "

	CmdInitDownloadAsk       = "It seems the init package could not be found locally, but can be downloaded from %s"
	CmdInitDownloadNote      = "Note: This will require an internet connection."
	CmdInitDownloadConfirm   = "Do you want to download this init package?"
	CmdInitDownloadCancel    = "Confirm selection canceled: %s"
	CmdInitDownloadErrManual = "download the init package manually and place it in the current working directory"

	CmdInitFlagConfirm      = "Confirm the install without prompting"
	CmdInitFlagComponents   = "Specify which optional components to install.  E.g. --components=git-server,logging"
	CmdInitFlagStorageClass = "Specify the storage class to use for the registry.  E.g. --storage-class=standard"

	CmdInitFlagGitURL      = "External git server url to use for this Zarf cluster"
	CmdInitFlagGitPushUser = "Username to access to the git server Zarf is configured to use. User must be able to create repositories via 'git push'"
	CmdInitFlagGitPushPass = "Password for the push-user to access the git server"
	CmdInitFlagGitPullUser = "Username for pull-only access to the git server"
	CmdInitFlagGitPullPass = "Password for the pull-only user to access the git server"

	CmdInitFlagRegRL       = "External registry url address to use for this Zarf cluster"
	CmdInitFlagRegNodePort = "Nodeport to access a registry internal to the k8s cluster. Between [30000-32767]"
	CmdInitFlagRegPushUser = "Username to access to the registry Zarf is configured to use"
	CmdInitFlagRegPushPass = "Password for the push-user to connect to the registry"
	CmdInitFlagRegPullUser = "Username for pull-only access to the registry"
	CmdInitFlagRegPullPass = "Password for the pull-only user to access the registry"
	CmdInitFlagRegSecret   = "Registry secret value"

	// zarf tools
	CmdToolsShort = "Collection of additional tools to make airgap easier"

	CmdToolsArchiverShort           = "Compress/Decompress generic archives, including Zar packages."
	CmdToolsArchiverCompressShort   = "Compress a collection of sources based off of the destination file extension."
	CmdToolsArchiverCompressErr     = "Unable to perform compression"
	CmdToolsArchiverDecompressShort = "Decompress an archive or Zarf package based off of the source file extension."
	CmdToolsArchiverDecompressErr   = "Unable to perform decompression"

	CmdToolsRegistryShort = "Tools for working with container registries using go-containertools."

	CmdToolsGetGitPasswdShort = "Returns the push user's password for the Git server"
	CmdToolsGetGitPasswdLong  = "Reads the password for a user with push access to the configured Git server from the zarf-state secret in the zarf namespace"
	CmdToolsGetGitPasswdInfo  = "Git Server Push Password: "

	CmdToolsMonitorShort = "Launch a terminal UI to monitor the connected cluster using K9s."

	CmdToolsClearCacheShort         = "Clears the configured git and image cache directory."
	CmdToolsClearCacheErr           = "Unable to clear the cache directory %s"
	CmdToolsClearCacheSuccess       = "Successfully cleared the cache from %s"
	CmdToolsClearCacheFlagCachePath = "Specify the location of the Zarf  artifact cache (images and git repositories)"

	CmdToolsGenPkiShort       = "Generates a Certificate Authority and PKI chain of trust for the given host"
	CmdToolsGenPkiSuccess     = "Successfully created a chain of trust for %s"
	CmdToolsGenPkiFlagAltName = "Specify Subject Alternative Names for the certificate"

	CmdToolsSbomShort = "Generates a Software Bill of Materials (SBOM) for the given package"
	CmdToolsSbomErr   = "Unable to create sbom (syft) CLI"

	// zarf version
	CmdVersionShort = "SBOM tools provided by Anchore Syft"
	CmdVersionLong  = "Displays the version of the Zarf release that the Zarf binary was built from."

	// cmd viper setup
	CmdViperErrLoadingConfigFile = "failed to load config file: %w"
	CmdViperInfoUsingConfigFile  = "Using config file %s"
)

// Zarf Agent messages
// These are only seen in the Kubernetes logs
const (
	AgentInfoWebhookAllowed = "Webhook [%s - %s] - Allowed: %t"
	AgentInfoShutdown       = "Shutdown gracefully..."
	AgentInfoPort           = "Server running in port: %s"

	AgentErrStart                  = "Failed to start the web server"
	AgentErrShutdown               = "unable to properly shutdown the web server"
	AgentErrNilReq                 = "malformed admission review: request is nil"
	AgentErrMarshalResponse        = "unable to marshal the response"
	AgentErrMarshallJSONPatch      = "unable to marshall the json patch"
	AgentErrInvalidType            = "only content type 'application/json' is supported"
	AgentErrInvalidOp              = "invalid operation: %s"
	AgentErrInvalidMethod          = "invalid method only POST requests are allowed"
	AgentErrImageSwap              = "Unable to swap the host for (%s)"
	AgentErrHostnameMatch          = "failed to complete hostname matching: %w"
	AgentErrGetState               = "failed to load zarf state from file: %w"
	AgentErrCouldNotDeserializeReq = "could not deserialize request: %s"
	AgentErrBindHandler            = "Unable to bind the webhook handler"
	AgentErrBadRequest             = "could not read request body: %s"
)

// ErrInitNotFound
var ErrInitNotFound = errors.New("this command requires a zarf-init package, but one was not found on the local system. Re-run the last command again without '--confirm' to download the package")
