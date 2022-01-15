package cmd

import (
	"fmt"
	"os"
	"regexp"

	"github.com/defenseunicorns/zarf/cli/internal/k8s"
	"github.com/defenseunicorns/zarf/cli/internal/message"
	"github.com/defenseunicorns/zarf/cli/internal/utils"

	"github.com/spf13/cobra"
)

var confirmDestroy bool

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Tear it all down, we'll miss you Zarf...",
	Run: func(cmd *cobra.Command, args []string) {
		burn()

		state := k8s.LoadZarfState()
		_ = os.Remove(".zarf-registry")

		if state.ZarfAppliance {
			// If Zarf deployed the cluster, burn it all down
			pattern := regexp.MustCompile(`(?mi)zarf-clean-.+\.sh$`)
			scripts := utils.RecursiveFileList("/usr/local/bin", pattern)
			// Iterate over al matching zarf-clean scripts and exec them
			for _, script := range scripts {
				// Run the matched script
				_, _ = utils.ExecCommand(true, nil, script)
				// Try to remove the script, but ignore any errors
				_ = os.Remove(script)
			}
		} else {
			// If Zarf didn't deploy the cluster, only delete the ZarfNamespace
			if err := k8s.DeleteZarfNamespace(); err != nil {
				message.Fatal(err, "Unable to fully remove the zarf namespace from this cluster")
			}
		}
		burn()
	},
}

func burn() {
	fmt.Println("")
	for count := 0; count < 40; count++ {
		fmt.Print("🔥")
	}
	fmt.Println("")
}

func init() {
	rootCmd.AddCommand(destroyCmd)

	destroyCmd.Flags().BoolVar(&confirmDestroy, "confirm", false, "Confirm the destroy action")
	_ = destroyCmd.MarkFlagRequired("confirm")
}
