// SPDX-FileCopyrightText: © 2025 Idiap Research Institute <contact@idiap.ch>
// SPDX-FileContributor: Samuel Gaist <samuel.gaist@idiap.ch>
//
// SPDX-License-Identifier: Apache-2.0

package pythoninstallers_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"

	pythoninstallers "github.com/paketo-buildpacks/python-package-managers-install"
	"github.com/paketo-buildpacks/python-package-managers-install/pkg/build"

	miniconda "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/miniconda"
	pip "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/pip"
	pipenv "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/pipenv"
	pixi "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/pixi"
	poetry "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/poetry"
	poetryfakes "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/poetry/fakes"
	uv "github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/uv"

	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		buffer     *bytes.Buffer

		parsePoetryProject *poetryfakes.PoetryPyProjectParser

		detect packit.DetectFunc

		plans []packit.BuildPlan
	)

	it.Before(func() {
		workingDir = t.TempDir()

		Expect(os.WriteFile(filepath.Join(workingDir, "x.py"), []byte{}, os.ModePerm)).To(Succeed())

		buffer = bytes.NewBuffer(nil)
		logger := scribe.NewEmitter(buffer)

		parsePoetryProject = &poetryfakes.PoetryPyProjectParser{}
		parsePoetryProject.ParsePythonVersionCall.Returns.String = "1.2.3"
		parsePoetryProject.IsPoetryProjectCall.Returns.Bool = true

		detect = pythoninstallers.Detect(logger, parsePoetryProject)

		plans = append(plans, packit.BuildPlan{
			Provides: []packit.BuildPlanProvision{
				{Name: pip.Pip},
			},
			Requires: []packit.BuildPlanRequirement{
				{
					Name: pip.CPython,
					Metadata: build.BuildPlanMetadata{
						Build: true,
					},
				},
			},
		},
		)

		plans = append(plans, packit.BuildPlan{
			Provides: []packit.BuildPlanProvision{
				{Name: miniconda.Conda},
			},
		},
		)

		plans = append(plans, packit.BuildPlan{
			Provides: []packit.BuildPlanProvision{
				{Name: pipenv.Pip},
				{Name: pipenv.Pipenv},
			},
			Requires: []packit.BuildPlanRequirement{
				{
					Name: pipenv.CPython,
					Metadata: build.BuildPlanMetadata{
						Build:  true,
						Launch: false,
					},
				},
				{
					Name: pipenv.Pip,
					Metadata: build.BuildPlanMetadata{
						Build:  true,
						Launch: false,
					},
				},
			},
		},
		)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("detection phase", func() {
		context("without pyproject.toml", func() {
			it("passes detection", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan).To(Equal(pythoninstallers.Or(plans...)))
			})
		})

		context("with pyproject.toml", func() {
			context("without build backend", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workingDir, "pyproject.toml"), []byte{}, os.ModePerm)).To(Succeed())
				})

				it("passes detection", func() {
					result, err := detect(packit.DetectContext{
						WorkingDir: workingDir,
					})

					withPoetry := append(plans,
						packit.BuildPlan{
							Provides: []packit.BuildPlanProvision{
								{Name: poetry.Pip},
								{Name: poetry.PoetryDependency},
							},
							Requires: []packit.BuildPlanRequirement{
								{
									Name: poetry.CPython,
									Metadata: build.BuildPlanMetadata{
										Build:         true,
										Version:       "1.2.3",
										VersionSource: "pyproject.toml",
									},
								},
								{
									Name: poetry.Pip,
									Metadata: build.BuildPlanMetadata{
										Build: true,
									},
								},
							},
						},
					)

					Expect(err).NotTo(HaveOccurred())
					Expect(result.Plan).To(Equal(pythoninstallers.Or(withPoetry...)))
				})
			})

			context("with poetry build backend", func() {
				it.Before(func() {
					content := []byte(`
					[build-system]
					requires = ["poetry-core>=1.0.0"]
					build-backend = "poetry.core.masonry.api"
					`)
					Expect(os.WriteFile(filepath.Join(workingDir, "pyproject.toml"), content, os.ModePerm)).To(Succeed())
					parsePoetryProject.IsPoetryProjectCall.Returns.Bool = true
				})

				it("passes detection", func() {
					result, err := detect(packit.DetectContext{
						WorkingDir: workingDir,
					})

					withPoetry := append(plans,
						packit.BuildPlan{
							Provides: []packit.BuildPlanProvision{
								{Name: poetry.Pip},
								{Name: poetry.PoetryDependency},
							},
							Requires: []packit.BuildPlanRequirement{
								{
									Name: poetry.CPython,
									Metadata: build.BuildPlanMetadata{
										Build:         true,
										Version:       "1.2.3",
										VersionSource: "pyproject.toml",
									},
								},
								{
									Name: poetry.Pip,
									Metadata: build.BuildPlanMetadata{
										Build: true,
									},
								},
							},
						},
					)

					Expect(err).NotTo(HaveOccurred())
					Expect(result.Plan).To(Equal(pythoninstallers.Or(withPoetry...)))
				})
			})

			context("with other build backend", func() {
				it.Before(func() {
					content := []byte(`
					[build-system]
					requires = ["setuptools", "setuptools-scm"]
					build-backend = "setuptools.build_meta"
					`)
					Expect(os.WriteFile(filepath.Join(workingDir, "pyproject.toml"), content, os.ModePerm)).To(Succeed())
					parsePoetryProject.IsPoetryProjectCall.Returns.Bool = false
				})

				it("passes detection", func() {
					result, err := detect(packit.DetectContext{
						WorkingDir: workingDir,
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(result.Plan).To(Equal(pythoninstallers.Or(plans...)))
				})
			})
		})

		context("with uv.lock", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, uv.LockfileName), []byte(`requires-python = "3.13.0"`), os.ModePerm)).To(Succeed())
			})

			it("passes detection", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})

				withUv := append(plans,
					packit.BuildPlan{
						Provides: []packit.BuildPlanProvision{
							{Name: uv.Uv},
						},
					},
				)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan).To(Equal(pythoninstallers.Or(withUv...)))
			})
		})

		context("with pixi.lock", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, pixi.LockfileName), []byte(""), os.ModePerm)).To(Succeed())
			})

			it("passes detection", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})

				withUv := append(plans,
					packit.BuildPlan{
						Provides: []packit.BuildPlanProvision{
							{Name: pixi.Pixi},
						},
					},
				)

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan).To(Equal(pythoninstallers.Or(withUv...)))
			})
		})

		context("When BP_ENABLE_PACKAGE_MANAGERS is set", func() {
			it.Before(func() {
				t.Setenv("BP_ENABLE_PACKAGE_MANAGERS", "true")
				Expect(os.WriteFile(filepath.Join(workingDir, "environment.yml"), []byte{}, os.ModePerm)).To(Succeed())
			})

			it("passes detection", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan.Provides).To(ContainElement(
					gstruct.MatchAllFields(gstruct.Fields{
						"Name": Equal(pythoninstallers.PackageManagersInstallPlanEntry),
					}),
				))
			})
		})

	})
}
