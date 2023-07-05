// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package oci contains functions for interacting with Zarf packages stored in OCI registries.
package oci

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/defenseunicorns/zarf/src/config"
	"github.com/defenseunicorns/zarf/src/pkg/message"
	"github.com/defenseunicorns/zarf/src/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
)

// Bundle pushes the given bundle to the remote repository.
func (o *OrasRemote) Bundle(bundle *types.ZarfBundle, sigPath string, sigPsswd string) error {
	message.Debug("Bundling", bundle.Metadata.Name, "to", o.Reference)
	layers := []ocispec.Descriptor{}
	index := ocispec.Index{}
	for _, pkg := range bundle.Packages {
		url := fmt.Sprintf("%s:%s-%s", pkg.Repository, pkg.Ref, bundle.Metadata.Architecture)
		remote, err := NewOrasRemote(url)
		if err != nil {
			return err
		}
		root, err := remote.FetchRoot()
		if err != nil {
			return err
		}
		manifestBytes, err := json.Marshal(root)
		if err != nil {
			return err
		}
		// push the manifest into the bundle
		manifestDesc, err := o.PushBytes(manifestBytes, ocispec.MediaTypeImageManifest)
		if err != nil {
			return err
		}
		// add the package name to the manifest's annotations to make it easier to find
		manifestDesc.Annotations = map[string]string{
			ocispec.AnnotationTitle: url,
		}
		index.Manifests = append(index.Manifests, manifestDesc)
		// stream copy the blobs from remote to o, otherwise do a blob mount
		if remote.Reference.Registry != o.Reference.Registry {
			message.Debugf("Streaming layers from %s --> %s", remote.Reference, o.Reference)
			if err := CopyPackage(remote, o, config.CommonOptions.OCIConcurrency); err != nil {
				return err
			}
		} else {
			message.Debugf("Performing a cross repository blob mount on %s from %s --> %s", remote.Reference.Registry, remote.Reference.Repository, o.Reference.Repository)
			for _, layer := range root.Layers {
				err := o.Mount(o.Context, layer, remote.Reference.Repository, func() (io.ReadCloser, error) {
					// TODO: how does this handle auth?
					return remote.Fetch(o.Context, layer)
				})
				if err != nil {
					return err
				}
			}
		}
		layers = append(layers, root.Layers...)
	}
	// push the index.json
	indexBytes, err := json.Marshal(index)
	if err != nil {
		return err
	}
	indexDesc, err := o.PushBytes(indexBytes, ocispec.MediaTypeImageIndex)
	if err != nil {
		return err
	}
	// when doing an oras.Copy, the annotation title is used as the filepath to download to
	indexDesc.Annotations = map[string]string{
		ocispec.AnnotationTitle: "index.json",
	}
	manifest := ocispec.Manifest{}
	manifest.Layers = layers
	for idx, layer := range manifest.Layers {
		// when doing an oras.Copy, the annotation title is used as the filepath to download to
		// this makes it so it downloads all of the package layers to blobs/sha256/<layer-digest>
		// just like a real OCI image
		manifest.Layers[idx].Annotations = map[string]string{
			ocispec.AnnotationTitle: filepath.Join("blobs", "sha256", layer.Digest.Encoded()),
		}
	}
	manifest.Layers = append(manifest.Layers, indexDesc)

	// TODO: push + append the zarf-bundle.yaml to the layers, w/ proper path
	// TODO: push + append the zarf-bundle.yaml.sig to the layers, w/ proper path
	message.Debug("TODO: signing bundle w/ %s - %s", sigPath, sigPsswd)

	manifest.Annotations = o.manifestAnnotationsFromMetadata(&bundle.Metadata)
	b, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	expected := content.NewDescriptorFromBytes(ocispec.MediaTypeImageManifest, b)

	_, err = o.pushManifestConfigFromMetadata(&bundle.Metadata, &bundle.Build)
	if err != nil {
		return err
	}

	return o.Manifests().PushReference(o.Context, expected, bytes.NewReader(b), o.Reference.Reference)
}
