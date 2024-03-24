---
title: zarf tools get-creds
description: Zarf CLI command reference for <code>zarf tools get-creds</code>.
tableOfContents: false
---

## zarf tools get-creds

Displays a table of credentials for deployed Zarf services. Pass a service key to get a single credential

### Synopsis

Display a table of credentials for deployed Zarf services. Pass a service key to get a single credential. i.e. 'zarf tools get-creds registry'

```
zarf tools get-creds [flags]
```

### Examples

```

# Print all Zarf credentials:
$ zarf tools get-creds

# Get specific Zarf credentials:
$ zarf tools get-creds registry
$ zarf tools get-creds registry-readonly
$ zarf tools get-creds git
$ zarf tools get-creds git-readonly
$ zarf tools get-creds artifact
$ zarf tools get-creds logging

```

### Options

```
  -h, --help   help for get-creds
```

### Options inherited from parent commands

```
  -a, --architecture string   Architecture for OCI images and Zarf packages
      --insecure              Allow access to insecure registries and disable other recommended security enforcements such as package checksum and signature validation. This flag should only be used if you have a specific reason and accept the reduced security posture.
  -l, --log-level string      Log level when running Zarf. Valid options are: warn, info, debug, trace (default "info")
      --no-color              Disable colors in output
      --no-log-file           Disable log file creation
      --no-progress           Disable fancy UI progress bars, spinners, logos, etc
      --tmpdir string         Specify the temporary directory to use for intermediate files
      --zarf-cache string     Specify the location of the Zarf cache directory (default "~/.zarf-cache")
```

### SEE ALSO

* [zarf tools](/commands/zarf_tools/)	 - Collection of additional tools to make airgap easier
