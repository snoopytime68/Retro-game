// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package sources contains core implementations of the PackageSource interface.
package sources

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/defenseunicorns/zarf/src/config"
	"github.com/defenseunicorns/zarf/src/pkg/message"
	"github.com/defenseunicorns/zarf/src/pkg/oci"
	"github.com/defenseunicorns/zarf/src/pkg/packager/layout"
	"github.com/defenseunicorns/zarf/src/pkg/utils"
	"github.com/defenseunicorns/zarf/src/pkg/utils/helpers"
	"github.com/defenseunicorns/zarf/src/types"
	"github.com/mholt/archiver/v3"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// OCISource is a package source for OCI registries.
type OCISource struct {
	*types.ZarfPackageOptions
	*oci.OrasRemote
}

// LoadPackage loads a package from an OCI registry.
func (s *OCISource) LoadPackage(dst *layout.PackagePaths) (err error) {
	var pkg types.ZarfPackage
	layersToPull := []ocispec.Descriptor{}

	message.Debugf("Loading package from %q", s.PackageSource)

	optionalComponents := helpers.StringToSlice(s.OptionalComponents)

	// only pull specified components and their images if optionalComponents AND --confirm are set
	if len(optionalComponents) > 0 && config.CommonOptions.Confirm {
		layers, err := s.LayersFromRequestedComponents(optionalComponents)
		if err != nil {
			return fmt.Errorf("unable to get published component image layers: %s", err.Error())
		}
		layersToPull = append(layersToPull, layers...)
	}

	isPartial := true
	root, err := s.FetchRoot()
	if err != nil {
		return err
	}
	if len(root.Layers) == len(layersToPull) {
		isPartial = false
	}

	layersFetched, err := s.PullPackage(dst.Base, config.CommonOptions.OCIConcurrency, layersToPull...)
	if err != nil {
		return fmt.Errorf("unable to pull the package: %w", err)
	}
	dst.SetFromLayers(layersFetched)

	if err := utils.ReadYaml(dst.ZarfYAML, &pkg); err != nil {
		return err
	}

	if err := ValidatePackageIntegrity(dst, pkg.Metadata.AggregateChecksum, isPartial); err != nil {
		return err
	}

	if err := ValidatePackageSignature(dst, s.PublicKeyPath); err != nil {
		return err
	}

	if err := LoadComponents(&pkg, dst); err != nil {
		return err
	}

	if err := dst.SBOMs.Unarchive(); err != nil {
		return err
	}

	return nil
}

// LoadPackageMetadata loads a package's metadata from an OCI registry.
func (s *OCISource) LoadPackageMetadata(dst *layout.PackagePaths, wantSBOM bool, skipValidation bool) (err error) {
	var pkg types.ZarfPackage

	toPull := oci.PackageAlwaysPull
	if wantSBOM {
		toPull = append(toPull, layout.SBOMTar)
	}

	layersFetched, err := s.PullPackagePaths(toPull, dst.Base)
	if err != nil {
		return err
	}
	dst.SetFromLayers(layersFetched)

	if utils.InvalidPath(dst.SBOMs.Path) && wantSBOM {
		return fmt.Errorf("package does not contain SBOMs")
	}

	if err := utils.ReadYaml(dst.ZarfYAML, &pkg); err != nil {
		return err
	}

	if err := ValidatePackageIntegrity(dst, pkg.Metadata.AggregateChecksum, true); err != nil {
		return err
	}

	if err := ValidatePackageSignature(dst, s.PublicKeyPath); err != nil {
		if errors.Is(err, ErrPkgSigButNoKey) && skipValidation {
			message.Warn("The package was signed but no public key was provided, skipping signature validation")
		} else {
			return err
		}
	}

	// unpack sboms.tar
	if err := dst.SBOMs.Unarchive(); err != nil {
		return err
	}

	return nil
}

// Collect pulls a package from an OCI registry and writes it to a tarball.
func (s *OCISource) Collect(dstTarball string) error {
	tmp, err := utils.MakeTempDir(config.CommonOptions.TempDirectory)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	_, err = s.PullPackage(tmp, config.CommonOptions.OCIConcurrency)
	if err != nil {
		return err
	}

	allTheLayers, err := filepath.Glob(filepath.Join(tmp, "*"))
	if err != nil {
		return err
	}

	_ = os.Remove(dstTarball)

	return archiver.Archive(allTheLayers, dstTarball)
}