// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package sources contains core implementations of the PackageSource interface.
package sources

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/defenseunicorns/zarf/src/pkg/message"
	"github.com/defenseunicorns/zarf/src/pkg/oci"
	"github.com/defenseunicorns/zarf/src/pkg/packager/layout"
	"github.com/defenseunicorns/zarf/src/pkg/utils"
	"github.com/defenseunicorns/zarf/src/types"
	"github.com/mholt/archiver/v3"
)

// TarballSource is a package source for tarballs.
type TarballSource struct {
	*types.ZarfPackageOptions
}

// LoadPackage loads a package from a tarball.
func (s *TarballSource) LoadPackage(dst *layout.PackagePaths) (err error) {
	var pkg types.ZarfPackage

	message.Debugf("Loading package from %q", s.PackageSource)

	pathsExtracted := []string{}

	err = archiver.Walk(s.PackageSource, func(f archiver.File) error {
		if f.IsDir() {
			return nil
		}
		header, ok := f.Header.(*tar.Header)
		if !ok {
			return fmt.Errorf("expected header to be *tar.Header but was %T", f.Header)
		}
		path := header.Name

		dir := filepath.Dir(path)
		if dir != "." {
			if err := os.MkdirAll(filepath.Join(dst.Base, dir), 0755); err != nil {
				return err
			}
		}

		dstPath := filepath.Join(dst.Base, path)
		pathsExtracted = append(pathsExtracted, path)
		dst, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		_, err = io.Copy(dst, f)
		if err != nil {
			return err
		}

		message.Debugf("Loaded %q --> %q", path, dstPath)
		return nil
	})
	if err != nil {
		return err
	}

	dst.SetFromPaths(pathsExtracted)

	if err := utils.ReadYaml(dst.ZarfYAML, &pkg); err != nil {
		return err
	}

	if err := ValidatePackageIntegrity(dst, pkg.Metadata.AggregateChecksum, false); err != nil {
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

// LoadPackageMetadata loads a package's metadata from a tarball.
func (s *TarballSource) LoadPackageMetadata(dst *layout.PackagePaths, wantSBOM bool, skipValidation bool) (err error) {
	var pkg types.ZarfPackage

	toExtract := oci.PackageAlwaysPull
	if wantSBOM {
		toExtract = append(toExtract, layout.SBOMTar)
	}
	pathsExtracted := []string{}

	for _, rel := range toExtract {
		if err := archiver.Extract(s.PackageSource, rel, dst.Base); err != nil {
			return err
		}
		// archiver.Extract will not return an error if the file does not exist, so we must manually check
		if !utils.InvalidPath(filepath.Join(dst.Base, rel)) {
			pathsExtracted = append(pathsExtracted, rel)
		}
	}

	dst.SetFromPaths(pathsExtracted)

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

// Collect for the TarballSource is essentially an `mv`
func (s *TarballSource) Collect(destinationTarball string) error {
	return os.Rename(s.PackageSource, destinationTarball)
}

// SplitTarballSource is a package source for split tarballs.
type SplitTarballSource struct {
	*types.ZarfPackageOptions
}

// Collect turns a split tarball into a full tarball.
func (s *SplitTarballSource) Collect(dstTarball string) error {
	pattern := strings.Replace(s.PackageSource, ".part000", ".part*", 1)
	fileList, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("unable to find split tarball files: %s", err)
	}

	// Ensure the files are in order so they are appended in the correct order
	sort.Strings(fileList)

	// Create the new package
	pkgFile, err := os.Create(dstTarball)
	if err != nil {
		return fmt.Errorf("unable to create new package file: %s", err)
	}
	defer pkgFile.Close()

	var pkgData types.ZarfSplitPackageData
	for idx, file := range fileList {
		// The first file contains metadata about the package
		if idx == 0 {
			var bytes []byte

			if bytes, err = os.ReadFile(file); err != nil {
				return fmt.Errorf("unable to read file %s: %w", file, err)
			}

			if err := json.Unmarshal(bytes, &pkgData); err != nil {
				return fmt.Errorf("unable to unmarshal file %s: %w", file, err)
			}

			count := len(fileList) - 1
			if count != pkgData.Count {
				return fmt.Errorf("package is missing parts, expected %d, found %d", pkgData.Count, count)
			}

			if len(s.Shasum) > 0 && pkgData.Sha256Sum != s.Shasum {
				return fmt.Errorf("mismatch in CLI options and package metadata, expected %s, found %s", s.Shasum, pkgData.Sha256Sum)
			}

			continue
		}

		// Open the file
		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("unable to open file %s: %w", file, err)
		}
		defer f.Close()

		// Add the file contents to the package
		if _, err = io.Copy(pkgFile, f); err != nil {
			return fmt.Errorf("unable to copy file %s: %w", file, err)
		}
	}

	if err := utils.SHAsMatch(dstTarball, pkgData.Sha256Sum); err != nil {
		return fmt.Errorf("package integrity check failed: %w", err)
	}

	// Remove the parts to reduce disk space before extracting
	for _, file := range fileList {
		_ = os.Remove(file)
	}

	// communicate to the user that the package was reassembled
	message.Infof("Reassembled package to: %q", dstTarball)

	return nil
}

// LoadPackage loads a package from a split tarball.
func (s *SplitTarballSource) LoadPackage(dst *layout.PackagePaths) (err error) {
	dstTarball := strings.Replace(s.PackageSource, ".part000", "", 1)

	if err := s.Collect(dstTarball); err != nil {
		return err
	}

	// Update the package source to the reassembled tarball
	s.PackageSource = dstTarball

	ts := &TarballSource{
		s.ZarfPackageOptions,
	}
	return ts.LoadPackage(dst)
}

// LoadPackageMetadata loads a package's metadata from a split tarball.
func (s *SplitTarballSource) LoadPackageMetadata(dst *layout.PackagePaths, wantSBOM bool, skipValidation bool) (err error) {
	dstTarball := strings.Replace(s.PackageSource, ".part000", "", 1)

	if err := s.Collect(dstTarball); err != nil {
		return err
	}

	// Update the package source to the reassembled tarball
	s.PackageSource = dstTarball

	ts := &TarballSource{
		s.ZarfPackageOptions,
	}
	return ts.LoadPackageMetadata(dst, wantSBOM, skipValidation)
}