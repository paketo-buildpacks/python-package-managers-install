// SPDX-FileCopyrightText: © 2025 Idiap Research Institute <contact@idiap.ch>
// SPDX-FileContributor: Samuel Gaist <samuel.gaist@idiap.ch>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"

	pythoninstallers "github.com/paketo-buildpacks/python-package-managers-install"
	"github.com/paketo-buildpacks/python-package-managers-install/pkg/build"
	miniconda "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/miniconda"
	pip "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/pip"
	pipenv "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/pipenv"
	pixi "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/pixi"
	poetry "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/poetry"
	uv "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/uv"
	"github.com/paketo-buildpacks/python-package-managers-install/pkg/sbom"
)

func main() {
	logger := scribe.NewEmitter(os.Stdout).WithLevel(os.Getenv("BP_LOG_LEVEL"))

	buildParameters := build.CommonBuildParameters{
		SbomGenerator: sbom.Generator{},
		Clock:         chronos.DefaultClock,
		Logger:        logger,
	}

	packagerParameters := map[string]pythoninstallers.PackagerParameters{
		miniconda.Conda: miniconda.CondaBuildParameters{
			DependencyManager: postal.NewService(cargo.NewTransport()),
			Runner:            miniconda.NewScriptRunner(pexec.NewExecutable("bash")),
		},
		pip.Pip: pip.PipBuildParameters{
			DependencyManager:  postal.NewService(cargo.NewTransport()),
			InstallProcess:     pip.NewPipInstallProcess(pexec.NewExecutable("python")),
			SitePackageProcess: pip.NewSiteProcess(pexec.NewExecutable("python")),
		},
		pipenv.Pipenv: pipenv.PipEnvBuildParameters{
			DependencyManager:  postal.NewService(cargo.NewTransport()),
			InstallProcess:     pipenv.NewPipenvInstallProcess(pexec.NewExecutable("python")),
			SitePackageProcess: pipenv.NewSiteProcess(pexec.NewExecutable("python")),
		},
		poetry.PoetryDependency: poetry.PoetryBuildParameters{
			DependencyManager:  postal.NewService(cargo.NewTransport()),
			InstallProcess:     poetry.NewPoetryInstallProcess(pexec.NewExecutable("python")),
			SitePackageProcess: poetry.NewSiteProcess(pexec.NewExecutable("python")),
		},
		uv.Uv: uv.UvBuildParameters{
			DependencyManager: postal.NewService(cargo.NewTransport()),
			InstallProcess:    uv.NewUvInstallProcess(),
		},
		pixi.Pixi: pixi.PixiBuildParameters{
			DependencyManager: postal.NewService(cargo.NewTransport()),
			InstallProcess:    pixi.NewPixiInstallProcess(),
		},
	}

	packit.Run(
		pythoninstallers.Detect(logger, poetry.NewPyProjectParser()),
		pythoninstallers.Build(logger, buildParameters, packagerParameters),
	)
}
