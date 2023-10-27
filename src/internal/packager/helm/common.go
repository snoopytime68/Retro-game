// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package helm contains operations for working with helm charts.
package helm

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/defenseunicorns/zarf/src/config"
	"github.com/defenseunicorns/zarf/src/internal/cluster"
	"github.com/defenseunicorns/zarf/src/pkg/message"
	"github.com/defenseunicorns/zarf/src/types"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
)

// Helm is a config object for working with helm charts.
type Helm struct {
	chart      types.ZarfChart
	chartPath  string
	valuesPath string

	cfg       *types.PackagerConfig
	component types.ZarfComponent
	cluster   *cluster.Cluster

	kubeVersion string

	chartOverride *chart.Chart
	valueOverride map[string]any

	settings     *cli.EnvSettings
	actionConfig *action.Configuration
}

// New returns a new Helm config struct.
func New(chart types.ZarfChart, chartPath string, valuesPath string) *Helm {
	return &Helm{
		chart:      chart,
		chartPath:  chartPath,
		valuesPath: valuesPath,
	}
}

// NewClusterOnly returns a new Helm config struct geared toward interacting with the cluster (not packages)
func NewClusterOnly(cfg *types.PackagerConfig, cluster *cluster.Cluster) *Helm {
	return &Helm{
		cfg:     cfg,
		cluster: cluster,
	}
}

// NewFromZarfManifest generates a helm chart and config from a given Zarf manifest.
func NewFromZarfManifest(manifest types.ZarfManifest, manifestPath, packageName, componentName string) (h *Helm, err error) {
	spinner := message.NewProgressSpinner("Starting helm chart generation %s", manifest.Name)
	defer spinner.Stop()

	// Generate a new chart.
	tmpChart := new(chart.Chart)
	tmpChart.Metadata = new(chart.Metadata)

	// Generate a hashed chart name.
	rawChartName := fmt.Sprintf("raw-%s-%s-%s", packageName, componentName, manifest.Name)
	hasher := sha1.New()
	hasher.Write([]byte(rawChartName))
	tmpChart.Metadata.Name = rawChartName
	sha1ReleaseName := hex.EncodeToString(hasher.Sum(nil))

	// This is fun, increment forward in a semver-way using epoch so helm doesn't cry.
	tmpChart.Metadata.Version = fmt.Sprintf("0.1.%d", config.GetStartTime())
	tmpChart.Metadata.APIVersion = chart.APIVersionV1

	// Add the manifest files so helm does its thing.
	for _, file := range manifest.Files {
		spinner.Updatef("Processing %s", file)
		manifest := path.Join(manifestPath, file)
		data, err := os.ReadFile(manifest)
		if err != nil {
			return h, fmt.Errorf("unable to read manifest file %s: %w", manifest, err)
		}

		// Escape all chars and then wrap in {{ }}.
		txt := strconv.Quote(string(data))
		data = []byte("{{" + txt + "}}")

		tmpChart.Templates = append(tmpChart.Templates, &chart.File{Name: manifest, Data: data})
	}

	// Generate the struct to pass to InstallOrUpgradeChart().
	h.chart = types.ZarfChart{
		Name: tmpChart.Metadata.Name,
		// Preserve the zarf prefix for chart names to match v0.22.x and earlier behavior.
		ReleaseName: fmt.Sprintf("zarf-%s", sha1ReleaseName),
		Version:     tmpChart.Metadata.Version,
		Namespace:   manifest.Namespace,
		NoWait:      manifest.NoWait,
	}
	h.chartOverride = tmpChart

	// We don't have any values because we do not expose them in the zarf.yaml currently.
	h.valueOverride = map[string]any{}

	spinner.Success()

	return h, nil
}

// WithDeployInfo adds the necessary information to deploy a given chart
func (h *Helm) WithDeployInfo(component types.ZarfComponent, cfg *types.PackagerConfig, cluster *cluster.Cluster) {
	h.component = component
	h.cfg = cfg
	h.cluster = cluster
}

// WithKubeVersion sets the Kube version for templating the chart
func (h *Helm) WithKubeVersion(kubeVersion string) {
	h.kubeVersion = kubeVersion
}

// StandardName generates a predictable full path for a helm chart for Zarf.
func StandardName(destination string, chart types.ZarfChart) string {
	return filepath.Join(destination, chart.Name+"-"+chart.Version)
}
