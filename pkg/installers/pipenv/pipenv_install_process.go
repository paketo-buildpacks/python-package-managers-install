// SPDX-FileCopyrightText: Copyright (c) 2013-Present CloudFoundry.org Foundation, Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package pipenv

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2/pexec"

	"github.com/paketo-buildpacks/python-package-managers-install/pkg/executable"
)

type PipenvInstallProcess struct {
	executable executable.Executable
}

// NewPipenvInstallProcess creates a PipenvInstallProcess instance.
func NewPipenvInstallProcess(executable executable.Executable) PipenvInstallProcess {
	return PipenvInstallProcess{
		executable: executable,
	}
}

// Execute installs the provided version of pipenv from the internet into the
// layer path designated by targetLayerPath
func (p PipenvInstallProcess) Execute(version, targetLayerPath, pipLayerPath string) error {
	buffer := bytes.NewBuffer(nil)

	pipPythonPath := fmt.Sprintf("PYTHONPATH=%s", filepath.Join(pipLayerPath))
	err := p.executable.Execute(pexec.Execution{
		Args: []string{"-m", "pip", "install", fmt.Sprintf("pipenv==%s", version), "--user"},
		// Set the PYTHONUSERBASE to ensure that pipenv is installed to the newly created target layer.
		Env:    append(os.Environ(), pipPythonPath, fmt.Sprintf("PYTHONUSERBASE=%s", targetLayerPath)),
		Stdout: buffer,
		Stderr: buffer,
	})

	if err != nil {
		return fmt.Errorf("failed to configure pipenv:\n%s\nerror: %w", buffer.String(), err)
	}

	return nil
}
