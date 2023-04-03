# Apono CLI
[![Actions Status](https://github.com/apono-io/apono-cli/workflows/CI/badge.svg?branch=main)](https://github.com/apono-io/apono-cli/actions?query=workflow%3ACI+branch%3Amain)

This repository provides a unified command line interface to Apono.

## Installation

### Using package manager (recommended)

#### MacOS and Linux using [Homebrew](https://brew.sh/)
```shell
brew tap apono-io/tap
brew install apono-cli
```

#### Windows using [Scoop](https://scoop.sh)
```powershell
scoop bucket add apono https://github.com/apono-io/scoop-bucket
scoop install apono/apono-cli
```

### Using pre-built releases

You can find pre-built releases of the CLI [here](https://github.com/apono-io/apono-cli/releases).

### From sources

To build `apono` from sources, a Go compiler >= 1.20 is required.

```shell
$ git clone https://github.com/apono-io/apono-cli
$ cd apono-cli
$ make all
```

Upon successful compilation, the resulting `apono` binary is stored in the `dist/` directory.
