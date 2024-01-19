// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package creator contains functions for creating Zarf packages.
package creator

import (
	"github.com/defenseunicorns/zarf/src/pkg/message"
	"github.com/defenseunicorns/zarf/src/pkg/packager/composer"
	"github.com/defenseunicorns/zarf/src/types"
)

func ComposeComponents(pkg *types.ZarfPackage, createOpts *types.ZarfCreateOptions) (composedPkg *types.ZarfPackage, warnings []string, err error) {
	components := []types.ZarfComponent{}

	pkgVars := pkg.Variables
	pkgConsts := pkg.Constants

	for i, component := range pkg.Components {
		arch := pkg.Build.Architecture
		// filter by architecture
		if !composer.CompatibleComponent(component, arch, createOpts.Flavor) {
			continue
		}

		// if a match was found, strip flavor and architecture to reduce bloat in the package definition
		component.Only.Cluster.Architecture = ""
		component.Only.Flavor = ""

		// build the import chain
		chain, err := composer.NewImportChain(component, i, pkg.Metadata.Name, arch, createOpts.Flavor)
		if err != nil {
			return nil, nil, err
		}
		message.Debugf("%s", chain)

		// migrate any deprecated component configurations now
		warnings = chain.Migrate(pkg.Build)

		// get the composed component
		composed, err := chain.Compose()
		if err != nil {
			return nil, nil, err
		}
		components = append(components, *composed)

		// merge variables and constants
		pkgVars = chain.MergeVariables(pkgVars)
		pkgConsts = chain.MergeConstants(pkgConsts)
	}

	// set the filtered + composed components
	pkg.Components = components

	pkg.Variables = pkgVars
	pkg.Constants = pkgConsts

	composedPkg = pkg

	return composedPkg, warnings, nil
}