package cmd

import (
	"shift/internal/k3s"
	"shift/internal/utils"

	"github.com/spf13/cobra"
)

var isDryRun bool

// initializeCmd represents the initialize command
var initializeCmd = &cobra.Command{
	Use:   "initialize",
	Short: "Deploys the utility cluster on a clean linux box",
	Long:  ` `,
	Run: func(cmd *cobra.Command, args []string) {

		utils.RunTarballChecksumValidate()
		utils.RunPreflightChecks()

		if !isDryRun {
			utils.PlaceAsset("bin/k3s", "/usr/local/bin/k3s")
			utils.PlaceAsset("bin/init-k3s.sh", "/usr/local/bin/init-k3s.sh")
			utils.PlaceAsset("charts", "/var/lib/rancher/k3s/server/static/charts")
			utils.PlaceAsset("manifests", "/var/lib/rancher/k3s/server/manifests")
			utils.PlaceAsset("images", "/var/lib/rancher/k3s/agent/images")

			// @todo: check for RHEL and install RPMs if available
			// yum localinstall -y --disablerepo=* --exclude container-selinux-1* TMP_PATH/rpms/*.rpm
			k3s.Install()
		}
	},
}

func init() {
	rootCmd.AddCommand(initializeCmd)
	initializeCmd.Flags().BoolVar(&isDryRun, "dryrun", false, "Only run checksum and preflight steps")
}
