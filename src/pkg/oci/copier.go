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

type ProgressTracker interface {
	Start(total int64, text string)
	Update(complete int64, text string)
	Current() int64
	Write([]byte) (int, error)
	Finish(format string, a ...any)
	Add(n int)
}

// Copy copies an artifact from one OCI registry to another
// ?! We should probably add in nil checks for progressTracker I assume
func Copy(ctx context.Context, src *OrasRemote, dst *OrasRemote,
	include func(d ocispec.Descriptor) bool, concurrency int, progressTracker ProgressTracker) error {
	// create a new semaphore to limit concurrency
	sem := semaphore.NewWeighted(int64(concurrency))

	// fetch the source root manifest
	srcRoot, err := src.FetchRoot()
	if err != nil {
		return err
	}

	var layers []ocispec.Descriptor
	for _, layer := range srcRoot.Layers {
		if include != nil && include(layer) {
			layers = append(layers, layer)
		} else if include == nil {
			layers = append(layers, layer)
		}
	}

	layers = append(layers, srcRoot.Config)

	size := int64(0)
	for _, layer := range layers {
		size += layer.Size
	}

	if progressTracker != nil {
		title := fmt.Sprintf("[0/%d] layers copied", len(layers))
		progressTracker.Start(size, title)
		defer progressTracker.Finish("Copied %s", src.repo.Reference)
	}

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
			if progressTracker != nil {
				progressTracker.Update(layer.Size, fmt.Sprintf("[%d/%d] layers copied", idx+1, len(layers)))
			}
			sem.Release(1)
			continue
		}

		eg, ectx := errgroup.WithContext(ctx)
		eg.SetLimit(2)

		var tr io.Reader
		var pw *io.PipeWriter

		// fetch the layer from the source
		rc, err := src.repo.Fetch(ectx, layer)
		if err != nil {
			return err
		}

		// ?! Does changing the order of this affect anything
		if progressTracker != nil {
			// create a new pipe so we can write to both the progressbar and the destination at the same time
			pr, pwTemp := io.Pipe()
			pw = pwTemp

			// TeeReader gets the data from the fetching layer and writes it to the PipeWriter
			tr = io.TeeReader(rc, pw)
			// this goroutine is responsible for updating the progressbar
			eg.Go(func() error {
				// read from the PipeReader to the progressbar
				if _, err := io.Copy(progressTracker, pr); err != nil {
					return fmt.Errorf("failed to update progress on layer %s: %w", layer.Digest, err)
				}
				return nil
			})
		} else {
			tr = rc
		}

		// this goroutine is responsible for pushing the layer to the destination
		eg.Go(func() error {
			defer func() {
				if pw != nil {
					pw.Close()
				}
			}()

			// get data from the TeeReader and push it to the destination
			// push the layer to the destination
			err = dst.repo.Push(ectx, layer, tr)
			if err != nil {
				return fmt.Errorf("failed to push layer %s to %s: %w", layer.Digest, dst.repo.Reference, err)
			}
			return nil
		})

		// wait for the goroutines to finish
		if err := eg.Wait(); err != nil {
			return err
		}
		sem.Release(1)
		if progressTracker != nil {
			progressTracker.Update(progressTracker.Current(), fmt.Sprintf("[%d/%d] layers copied", idx+1, len(layers)))
		}
	}

	duration := time.Since(start)
	src.log("Copied", src.repo.Reference, "to", dst.repo.Reference, "with a concurrency of", concurrency, "and took", duration)

	return nil
}
