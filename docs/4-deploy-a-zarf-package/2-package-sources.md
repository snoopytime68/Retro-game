# Package Sources

Zarf supports interacting with Zarf packages from a variety of sources. For library users of Zarf looking to implement their own, please refer to the `PackageSource` interface in `src/pkg/packager/sources/new.go`.

> all commands below are assumed to be prefixed w/ `zarf package`
>
> By saying a source is supported on `publish`, this means that the source is supported as the first argument to `publish`. Currently, only OCI references are supported as the second argument to `publish`.

## Core + Internally Supported

### Local Tarball

> Supported on: `create`, `deploy`, `inspect`, `remove`, `pull`, `mirror-resources`

The default result of a `create`.

Satisfied by: `zarf-package-*.tar` or `zarf-package-*.tar.zst`.

> Note that Zarf currently only supports `tar` and `tar.zst` as archival/compression formats.

Examples:

```text
zarf-init-amd64-v0.29.2.tar.zst
zarf-package-argocd-amd64.tar.zst
zarf-package-dos-games-amd64-1.0.0.tar
../zarf-package-manifests-amd64-0.0.1.tar.zst
some-dir/zarf-package-yolo-arm64.tar.zst
```

- `zarf package create <dir> ...` results in a local `.tar.zst` in `<dir>`
- `zarf package deploy <tarball> ...`
- `zarf package inspect <tarball> ...`
- `zarf package remove <tarball> ...`
- `zarf package publish <tarball> oci://<oci-ref> ...`
- `zarf package pull <tarball> <dir> ...` (this is essentially a `cp` to `<dir>`)
- `zarf package mirror-resources <tarball> ...`

### OCI Reference

> Supported on: `create`, `deploy`, `inspect`, `remove`, `publish`, `pull`, `mirror-resources`

<!-- TODO: successive operations will be a cache-first approach in the future, update this doc when that occurs -->

### HTTP(S) URL

> Supported on: `deploy`, `inspect`, `remove`, `publish`, `pull`, `mirror-resources`

### Split Tarball

> Supported on: `create`, `deploy`, `inspect`, `remove`, `publish`, `pull`, `mirror-resources`

## Specialized

### In-Cluster

> Supported on: `inspect`, `remove`