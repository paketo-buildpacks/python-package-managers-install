// SPDX-FileCopyrightText: Copyright (c) 2013-Present CloudFoundry.org Foundation, Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package pipenv_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/python-package-managers-install/pkg/executable/fakes"
	"github.com/paketo-buildpacks/python-package-managers-install/pkg/installers/pipenv"

	. "github.com/onsi/gomega"
)

func testPipenvInstallProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		version       = "1.2.3-some.version"
		destLayerPath string
		pipLayerPath  string
		executable    *fakes.Executable

		pipenvInstallProcess pipenv.PipenvInstallProcess
	)

	it.Before(func() {
		destLayerPath = t.TempDir()
		pipLayerPath = t.TempDir()

		executable = &fakes.Executable{}

		pipenvInstallProcess = pipenv.NewPipenvInstallProcess(executable)
	})

	context("Execute", func() {
		context("there is a pipenv dependency to install", func() {
			it("installs it to the pipenv layer", func() {
				err := pipenvInstallProcess.Execute(version, destLayerPath, pipLayerPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution.Env).To(Equal(append(os.Environ(),
					fmt.Sprintf("PYTHONPATH=%s", pipLayerPath),
					fmt.Sprintf("PYTHONUSERBASE=%s", destLayerPath)),
				))
				Expect(executable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{"-m", "pip", "install", "pipenv==1.2.3-some.version", "--user"}))
			})
		})

		context("failure cases", func() {
			context("the install process fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						_, err := fmt.Fprintln(execution.Stdout, "stdout output")
						Expect(err).NotTo(HaveOccurred())
						_, err = fmt.Fprintln(execution.Stderr, "stderr output")
						Expect(err).NotTo(HaveOccurred())
						return errors.New("installing pipenv failed")
					}
				})

				it("returns an error", func() {
					err := pipenvInstallProcess.Execute(version, destLayerPath, pipLayerPath)
					Expect(err).To(MatchError(ContainSubstring("installing pipenv failed")))
					Expect(err).To(MatchError(ContainSubstring("stdout output")))
					Expect(err).To(MatchError(ContainSubstring("stderr output")))
				})
			})
		})
	})
}
