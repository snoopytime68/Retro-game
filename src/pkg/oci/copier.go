// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package oci contains functions for interacting with Zarf packages stored in OCI registries.
package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// Copy copies an artifact from one OCI registry to another
func Copy(ctx context.Context, src *OrasRemote, dst *OrasRemote,
	include func(d ocispec.Descriptor) bool, concurrency int, progressBar ProgressWriter) (err error) {
	if progressBar == nil {
		progressBar = DiscardProgressWriter{}
	}
	// create a new semaphore to limit concurrency
	sem := semaphore.NewWeighted(int64(concurrency))

	// fetch the source root manifest
	srcRoot, err := src.FetchRoot()
	if err != nil {
		return err
	}

	layers := srcRoot.GetLayers(include)

	start := time.Now()

	for idx, layer := range layers {
		bytes, err := json.MarshalIndent(layer, "", "  ")
		if err != nil {
			src.log("ERROR marshalling json: %s", err.Error())
		}
		src.log("Copying layer:", string(bytes))
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		// check if the layer already exists in the destination
		exists, err := dst.repo.Exists(ctx, layer)
		if err != nil {
			return err
		}
		if exists {
			src.log("Layer already exists in destination, skipping")
			contentLength := layer.Size
			data := make([]byte, contentLength)
			progressBar.Write(data)
			progressBar.UpdateTitle(fmt.Sprintf("[%d/%d] layers copied", idx+1, len(layers)))
			sem.Release(1)
			continue
		}

		eg, ectx := errgroup.WithContext(ctx)
		eg.SetLimit(2)

		// fetch the layer from the source
		rc, err := src.repo.Fetch(ectx, layer)
		if err != nil {
			return err
		}

		// create a new pipe so we can write to both the progressbar and the destination at the same time
		pr, pw := io.Pipe()

		// TeeReader gets the data from the fetching layer and writes it to the PipeWriter
		tr := io.TeeReader(rc, pw)

		// this goroutine is responsible for pushing the layer to the destination
		eg.Go(func() error {
			defer pw.Close()

			// get data from the TeeReader and push it to the destination
			// push the layer to the destination
			err = dst.repo.Push(ectx, layer, tr)
			if err != nil {
				return fmt.Errorf("failed to push layer %s to %s: %w", layer.Digest, dst.repo.Reference, err)
			}
			return nil
		})

		// this goroutine is responsible for updating the progressbar
		eg.Go(func() error {
			// read from the PipeReader to the progressbar
			if _, err := io.Copy(progressBar, pr); err != nil {
				return fmt.Errorf("failed to update progress on layer %s: %w", layer.Digest, err)
			}
			return nil
		})

		// wait for the goroutines to finish
		if err := eg.Wait(); err != nil {
			return err
		}
		sem.Release(1)
		progressBar.UpdateTitle(fmt.Sprintf("[%d/%d] layers copied", idx+1, len(layers)))
	}

	duration := time.Since(start)
	src.log("Copied", src.repo.Reference, "to", dst.repo.Reference, "with a concurrency of", concurrency, "and took", duration)

	return nil
}
