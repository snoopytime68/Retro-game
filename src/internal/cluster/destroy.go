package cluster

import (
	"context"
	"errors"
	"os"
	"regexp"

	"github.com/defenseunicorns/zarf/src/config"
	"github.com/defenseunicorns/zarf/src/internal/helm"
	"github.com/defenseunicorns/zarf/src/internal/k8s"
	"github.com/defenseunicorns/zarf/src/internal/message"
	"github.com/defenseunicorns/zarf/src/internal/utils"
)

func DestroyZarfCluster(removeComponents bool) {
	// NOTE: If 'zarf init' failed to deploy the k3s component (or if we're looking at the wrong kubeconfig)
	//       there will be no zarf-state to load and the struct will be empty. In these cases, if we can find
	//       the scripts to remove k3s, we will still try to remove a locally installed k3s cluster
	state := k8s.LoadZarfState()

	// If Zarf deployed the cluster, burn it all down
	if state.ZarfAppliance || (state.Secret == "") {
		// Check if we have the scripts to destory everything
		fileInfo, err := os.Stat(config.ZarfCleanupScriptsPath)
		if errors.Is(err, os.ErrNotExist) || !fileInfo.IsDir() {
			message.Warnf("Unable to find the folder (%#v) which has the scripts to cleanup the cluster. Do you have the right kube-context?\n", config.ZarfCleanupScriptsPath)
			return
		}

		// Run all the scripts!
		pattern := regexp.MustCompile(`(?mi)zarf-clean-.+\.sh$`)
		scripts := utils.RecursiveFileList(config.ZarfCleanupScriptsPath, pattern)
		// Iterate over all matching zarf-clean scripts and exec them
		for _, script := range scripts {
			// Run the matched script
			_, _, err := utils.ExecCommandWithContext(context.TODO(), true, script)
			if errors.Is(err, os.ErrPermission) {
				message.Warnf("Got a 'permission denied' when trying to execute the script (%s). Are you the right user and/or do you have the right kube-context?\n", script)

				// Don't remove scripts we can't execute so the user can try to manually run
				continue
			} else if err != nil {
				message.Debugf("Received error when trying to execute the script (%s): %#v", script, err)
			}

			// Try to remove the script, but ignore any errors
			_ = os.Remove(script)
		}
	} else {
		// Perform chart uninstallation
		helm.Destroy(removeComponents)

		// If Zarf didn't deploy the cluster, only delete the ZarfNamespace
		k8s.DeleteZarfNamespace()

		// Remove zarf agent labels and secrets from namespaces Zarf doesn't manage
		k8s.StripZarfLabelsAndSecretsFromNamespaces()
	}
}
