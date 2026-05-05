// SPDX-FileCopyrightText: © 2025 Idiap Research Institute <contact@idiap.ch>
// SPDX-FileContributor: Samuel Gaist <samuel.gaist@idiap.ch>
//
// SPDX-License-Identifier: Apache-2.0

package pythoninstallers

import (
	"fmt"
	"os"
	"strconv"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"

	miniconda "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/miniconda"
	pip "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/pip"
	pipenv "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/pipenv"
	pixi "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/pixi"
	poetry "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/poetry"
	uv "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/uv"
)

// Detect will return a packit.DetectFunc that will be invoked during the
// detect phase of the buildpack lifecycle.
//
// If this buildpack detects files that indicate your app is a Python project,
// it will pass detection.
func Detect(logger scribe.Emitter, pyProjectParser poetry.PyProjectParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		plans := []packit.BuildPlan{}

		pipResult, err := pip.Detect()(context)

		if err == nil {
			plans = append(plans, pipResult.Plan)
		} else {
			logger.Detail("%s", err)
		}

		minicondaResult, err := miniconda.Detect()(context)

		if err == nil {
			plans = append(plans, minicondaResult.Plan)
		} else {
			logger.Detail("%s", err)
		}

		pipenvResult, err := pipenv.Detect()(context)

		if err == nil {
			plans = append(plans, pipenvResult.Plan)
		} else {
			logger.Detail("%s", err)
		}

		poetryResult, err := poetry.Detect(pyProjectParser)(context)

		if err == nil {
			plans = append(plans, poetryResult.Plan)
		} else {
			logger.Detail("%s", err)
		}

		uvResult, err := uv.Detect()(context)

		if err == nil {
			plans = append(plans, uvResult.Plan)
		} else {
			logger.Detail("%s", err)
		}

		pixiResult, err := pixi.Detect()(context)

		if err == nil {
			plans = append(plans, pixiResult.Plan)
		} else {
			logger.Detail("%s", err)
		}

		if len(plans) == 0 {
			return packit.DetectResult{}, packit.Fail.WithMessage("No python packager manager related files found")
		}

		shouldUsePackageManagers := false

		if usePackageManagers, ok := os.LookupEnv(PackageManagersEnv); ok {
			shouldUsePackageManagers, err = strconv.ParseBool(usePackageManagers)
			if err != nil {
				return packit.DetectResult{}, fmt.Errorf("failed to parse %s value %s: %w", PackageManagersEnv, usePackageManagers, err)
			}
		}

		if shouldUsePackageManagers {
			for i, plan := range plans {
				plans[i].Provides = append(plan.Provides, packit.BuildPlanProvision{
					Name: PackageManagersInstallPlanEntry,
				})
			}
		}

		return packit.DetectResult{
			Plan: Or(plans...),
		}, nil
	}
}

func Or(plans ...packit.BuildPlan) packit.BuildPlan {
	if len(plans) < 1 {
		return packit.BuildPlan{}
	}
	combinedPlan := plans[0]

	for i := range plans {
		if i == 0 {
			continue
		}
		combinedPlan.Or = append(combinedPlan.Or, plans[i])
	}
	return combinedPlan
}
