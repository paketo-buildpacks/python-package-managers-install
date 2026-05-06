<!--
SPDX-FileCopyrightText: © 2025 Idiap Research Institute <contact@idiap.ch>
SPDX-FileContributor: Samuel Gaist <samuel.gaist@idiap.ch>

SPDX-License-Identifier: Apache-2.0
-->

# Python Installers Cloud Native Buildpack

The Paketo Buildpack for Python Installers is a Cloud Native Buildpack that
installs python package managers.

The buildpack is published for consumption at
`gcr.io/paketo-buildpacks/python-package-managers-install` and
`paketobuildpacks/python-package-managers-install`.

## Behavior
This buildpack participates if one of the following detection succeeds:

- [miniconda](pkg/installers/miniconda/README.md) -> Always
- [pip](pkg/installers/pip/README.md) -> Always
- [pipenv](pkg/installers/pipenv/README.md) -> Always
- [poetry](pkg/installers/poetry/README.md) -> `pyproject.toml` is present in the root folder
- [uv](pkg/installers/uv/README.md) -> `uv.lock` is present in the root folder
- [pixi](pkg/installers/pixi/README.md) -> `pixi.lock` is present in the root folder

The buildpack will do the following:
* At build time:
  - Installs the package manager
  - Makes it available on the `PATH`
  - Adjusts `PYTHONPATH` as required
* At run time:
  - Does nothing

## Configuration

### `BP_ENABLE_PACKAGE_MANAGERS`

The `BP_ENABLE_PACKAGE_MANAGERS` environment variable allows you to force the use
of this buildpack for all the supported package managers. It works in tandem
with `python-start`. `python-start` will add a requirement that is fulfilled by
this buildpack.

It is currently used as an opt-in to allow Paketo users to do tests before the
old buildpacks get retired.

```shell
BP_ENABLE_PACKAGE_MANAGERS=true
```

## Usage

To package this buildpack for consumption:
```
$ ./scripts/package.sh --version x.x.x
```
This will create a `buildpackage.cnb` file under the build directory which you
can use to build your app as follows: `pack build <app-name> -p <path-to-app>
-b <cpython buildpack> -b <pip buildpack> -b build/buildpackage.cnb -b
<other-buildpacks..>`.

To run the unit and integration tests for this buildpack:
```
$ ./scripts/unit.sh && ./scripts/integration.sh
```
