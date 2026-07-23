// SPDX-FileCopyrightText: © 2025 Idiap Research Institute <contact@idiap.ch>
// SPDX-FileContributor: Samuel Gaist <samuel.gaist@idiap.ch>
//
// SPDX-License-Identifier: Apache-2.0

// Part of this code is taken from their respective buildpacks

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v81/github"
	"github.com/joshuatcasey/libdependency/buildpack_config"
	"github.com/joshuatcasey/libdependency/retrieve"
	"github.com/joshuatcasey/libdependency/upstream"
	"github.com/joshuatcasey/libdependency/versionology"
	"github.com/nfx/go-htmltable"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/fs"
)

type PyPiProductMetadataRaw struct {
	Releases map[string][]struct {
		PackageType string            `json:"packagetype"`
		URL         string            `json:"url"`
		UploadTime  string            `json:"upload_time_iso_8601"`
		Digests     map[string]string `json:"digests"`
	} `json:"releases"`
}

type PyPiRelease struct {
	version      *semver.Version
	SourceURL    string
	UploadTime   time.Time
	SourceSHA256 string
}

func (release PyPiRelease) Version() *semver.Version {
	return release.version
}

func getAllVersionsForInstaller(installer string) retrieve.GetAllVersionsFunc {
	fmt.Printf("Handling: %s\n", installer)
	if installer == "miniconda3" {
		return getAllMinicondaVersions
	}

	if installer == "uv" {
		return getAllUvVersions
	}

	if installer == "pixi" {
		return getAllPixiVersions
	}

	return func() (versionology.VersionFetcherArray, error) {

		var pypiMetadata PyPiProductMetadataRaw
		err := upstream.GetAndUnmarshal(fmt.Sprintf("https://pypi.org/pypi/%s/json", installer), &pypiMetadata)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve new versions from upstream: %w", err)
		}

		var allVersions versionology.VersionFetcherArray

		for version, releasesForVersion := range pypiMetadata.Releases {
			for _, release := range releasesForVersion {
				if release.PackageType != "sdist" {
					continue
				}

				fmt.Printf("Parsing semver version %s\n", version)

				newVersion, err := semver.NewVersion(version)
				if err != nil {
					continue
				}

				uploadTime, err := time.Parse(time.RFC3339, release.UploadTime)
				if err != nil {
					return nil, fmt.Errorf("could not parse upload time '%s' as date for version %s: %w", release.UploadTime, version, err)
				}

				allVersions = append(allVersions, PyPiRelease{
					version:      newVersion,
					SourceSHA256: release.Digests["sha256"],
					SourceURL:    release.URL,
					UploadTime:   uploadTime,
				})
			}
		}

		return allVersions, nil
	}
}

func generatePipMetadata(versionFetcher versionology.VersionFetcher) ([]versionology.Dependency, error) {
	version := versionFetcher.Version().String()
	pipRelease, ok := versionFetcher.(PyPiRelease)
	if !ok {
		return nil, errors.New("expected a PyPiRelease")
	}

	configMetadataDependency := cargo.ConfigMetadataDependency{
		CPE:            fmt.Sprintf("cpe:2.3:a:pypa:pip:%s:*:*:*:*:python:*:*", version),
		ID:             "pip",
		Licenses:       retrieve.LookupLicenses(pipRelease.SourceURL, upstream.DefaultDecompress),
		PURL:           retrieve.GeneratePURL("pip", version, pipRelease.SourceSHA256, pipRelease.SourceURL),
		Source:         pipRelease.SourceURL,
		SourceChecksum: fmt.Sprintf("sha256:%s", pipRelease.SourceSHA256),
		Stacks:         []string{"*"},
		Version:        version,
	}

	return versionology.NewDependencyArray(configMetadataDependency, "noarch")
}

func generatePipenvMetadata(versionFetcher versionology.VersionFetcher) ([]versionology.Dependency, error) {
	version := versionFetcher.Version().String()
	pipenvRelease, ok := versionFetcher.(PyPiRelease)
	if !ok {
		return nil, errors.New("expected a PyPiRelease")
	}

	configMetadataDependency := cargo.ConfigMetadataDependency{
		CPE:            fmt.Sprintf("cpe:2.3:a:python-pipenv:pipenv:%s:*:*:*:*:python:*:*", version),
		Checksum:       fmt.Sprintf("sha256:%s", pipenvRelease.SourceSHA256),
		ID:             "pipenv",
		Licenses:       retrieve.LookupLicenses(pipenvRelease.SourceURL, upstream.DefaultDecompress),
		Name:           "Pipenv",
		PURL:           retrieve.GeneratePURL("pipenv", version, pipenvRelease.SourceSHA256, pipenvRelease.SourceURL),
		Source:         pipenvRelease.SourceURL,
		SourceChecksum: fmt.Sprintf("sha256:%s", pipenvRelease.SourceSHA256),
		Stacks:         []string{"*"},
		URI:            pipenvRelease.SourceURL,
		Version:        version,
	}

	return []versionology.Dependency{{
		ConfigMetadataDependency: configMetadataDependency,
		SemverVersion:            versionFetcher.Version(),
	}}, nil
}

func generatePoetryMetadata(versionFetcher versionology.VersionFetcher) ([]versionology.Dependency, error) {
	version := versionFetcher.Version().String()
	poetryRelease, ok := versionFetcher.(PyPiRelease)
	if !ok {
		return nil, errors.New("expected a PyPiRelease")
	}

	configMetadataDependency := cargo.ConfigMetadataDependency{
		CPE:            fmt.Sprintf("cpe:2.3:a:python-poetry:poetry:%s:*:*:*:*:python:*:*", version),
		Checksum:       fmt.Sprintf("sha256:%s", poetryRelease.SourceSHA256),
		ID:             "poetry",
		Licenses:       retrieve.LookupLicenses(poetryRelease.SourceURL, upstream.DefaultDecompress),
		Name:           "Poetry",
		PURL:           retrieve.GeneratePURL("poetry", version, poetryRelease.SourceSHA256, poetryRelease.SourceURL),
		Source:         poetryRelease.SourceURL,
		SourceChecksum: fmt.Sprintf("sha256:%s", poetryRelease.SourceSHA256),
		Stacks:         []string{"*"},
		URI:            poetryRelease.SourceURL,
		Version:        version,
	}

	return []versionology.Dependency{{
		ConfigMetadataDependency: configMetadataDependency,
		SemverVersion:            versionFetcher.Version(),
	}}, nil
}

type MinicondaRelease struct {
	version      *semver.Version
	fullVersion  *semver.Version
	Arch         string
	SourceURL    string
	UploadTime   time.Time
	SourceSHA256 string
	BinaryURL    string
	BinarySHA256 string
}

func (release MinicondaRelease) Version() *semver.Version {
	return release.version
}

func (release MinicondaRelease) FullVersion() *semver.Version {
	return release.fullVersion
}

type Miniconda struct {
	Filename     string `header:"Filename"`
	Size         int    `header:"Size"`
	LastModified string `header:"Last Modified"`
	SHA256       string `header:"SHA256"`
}

const MaxNumberOfMinicondaReleases = 8

var ArchMap = map[string]string{
	"x86_64":  "amd64",
	"aarch64": "arm64",
}

// filtered returns the slice passed in parameter with the needle removed
func filtered(haystack []Miniconda, needle *regexp.Regexp) []Miniconda {
	output := []Miniconda{}

	for _, entry := range haystack {
		if needle.MatchString(entry.Filename) {
			output = append(output, entry)
		}
	}

	return output
}

func getAllMinicondaVersions() (versionology.VersionFetcherArray, error) {
	url := "https://repo.anaconda.com/miniconda"
	condaInstallersTable, err := htmltable.NewSliceFromURL[Miniconda](url)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`Miniconda3-py310_(?P<Version>\d+.\d+.\d+(-\d+)?)-Linux-(?P<Arch>x86_64|aarch64)`)
	matches := filtered(condaInstallersTable, re)

	// Keeping only six most recent versions as otherwise it means spamming the release
	// page from GitHub for getting the sha256 for the sources which will get throttled
	kept := matches[:MaxNumberOfMinicondaReleases]

	// var allVersions versionology.VersionFetcherArray
	// var allVersions []MinicondaRelease
	filteredVersions := map[string]MinicondaRelease{}

	for _, item := range kept {
		matches := re.FindStringSubmatch(item.Filename)

		fmt.Printf("Parsing semver version %s\n", matches[1])

		fullVersion, err := semver.NewVersion(matches[1])
		if err != nil {
			continue
		}

		originalVersion := strings.TrimSuffix(matches[1], matches[2])
		newVersion, err := semver.NewVersion(originalVersion)
		if err != nil {
			continue
		}

		versionString := fmt.Sprintf("%s-%s", newVersion.String(), matches[3])
		if knownVersion, ok := filteredVersions[versionString]; ok {
			knownPrerelease, err := strconv.Atoi(knownVersion.FullVersion().Prerelease())
			if err != nil {
				return nil, fmt.Errorf("could not parse prerelease value %s: %s", knownVersion.FullVersion().Prerelease(), err)
			}
			newPrerelease, err := strconv.Atoi(fullVersion.Prerelease())
			if err != nil {
				return nil, fmt.Errorf("could not parse prerelease value %s: %s", fullVersion.Prerelease(), err)
			}

			if knownPrerelease > newPrerelease {
				fmt.Println("skip", fullVersion, "as more recent version (", knownVersion.FullVersion(), ") was already found")
				continue
			}
		}

		uploadTime, err := time.Parse(time.DateTime, item.LastModified)
		if err != nil {
			return nil, fmt.Errorf("could not parse upload time '%s' as date for version %s: %w", item.LastModified, fullVersion, err)
		}

		sha256URL := fmt.Sprintf("https://github.com/conda/conda/releases/download/%s/conda-%s.tar.gz.sha256sum", originalVersion, originalVersion)
		resp, err := http.Get(sha256URL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		sourceSHA256, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read body of sha256 request")
		}

		sourceSHA256 = bytes.TrimSpace(sourceSHA256)

		if len(sourceSHA256) != sha256.BlockSize {
			return nil, fmt.Errorf("source sha256 does not have the correct size: %d vs %d", len(sourceSHA256), sha256.BlockSize)
		}

		filteredVersions[versionString] = MinicondaRelease{
			version:      newVersion,
			fullVersion:  fullVersion,
			Arch:         ArchMap[matches[3]],
			BinaryURL:    fmt.Sprintf("%s/%s", url, item.Filename),
			BinarySHA256: item.SHA256,
			SourceURL:    fmt.Sprintf("https://github.com/conda/conda/releases/download/%s/conda-%s.tar.gz", originalVersion, originalVersion),
			SourceSHA256: string(sourceSHA256),
			UploadTime:   uploadTime,
		}
	}

	var result versionology.VersionFetcherArray
	for _, version := range filteredVersions {
		result = append(result, version)
	}
	return result, nil
}

func generateMinicondaMetadata(versionFetcher versionology.VersionFetcher) ([]versionology.Dependency, error) {
	version := versionFetcher.Version().String()
	minicondaRelease, ok := versionFetcher.(MinicondaRelease)
	if !ok {
		return nil, errors.New("expected a MinicondaRelease")
	}

	var licenseIDsAsInterface []interface{}
	licenseIDsAsInterface = append(licenseIDsAsInterface, "BSD-3-Clause")
	configMetadataDependency := cargo.ConfigMetadataDependency{
		CPE:            fmt.Sprintf("cpe:2.3:a:conda:miniconda3:%s:*:*:*:*:python:*:*", version),
		Checksum:       fmt.Sprintf("sha256:%s", minicondaRelease.BinarySHA256),
		ID:             "miniconda3",
		Licenses:       licenseIDsAsInterface,
		Name:           "Miniconda.sh",
		OS:             "linux",
		Arch:           minicondaRelease.Arch,
		PURL:           retrieve.GeneratePURL("miniconda3", version, minicondaRelease.SourceSHA256, minicondaRelease.SourceURL),
		Source:         minicondaRelease.SourceURL,
		SourceChecksum: fmt.Sprintf("sha256:%s", minicondaRelease.SourceSHA256),
		Stacks:         []string{"*"},
		URI:            minicondaRelease.BinaryURL,
		Version:        version,
	}

	return []versionology.Dependency{{
		ConfigMetadataDependency: configMetadataDependency,
		SemverVersion:            versionFetcher.Version(),
	}}, nil
}

type GitHubRelease struct {
	version      *semver.Version
	Arch         string
	SourceURL    string
	UploadTime   time.Time
	SourceSHA256 string
	BinaryURL    string
	BinarySHA256 string
}

func (release GitHubRelease) Version() *semver.Version {
	return release.version
}

func getGitHubVersions(org string, project string, archAsset string) (versionology.VersionFetcherArray, error) {
	client := github.NewClient(nil)

	opt := &github.ListOptions{Page: 1, PerPage: 4}
	releases, _, err := client.Repositories.ListReleases(context.Background(), org, project, opt)

	if err != nil {
		return nil, err
	}

	var result versionology.VersionFetcherArray

	for _, release := range releases {
		version, err := semver.NewVersion(*release.TagName)
		if err != nil {
			return nil, err
		}

		var sourceURL *string
		var sourceSHA256 *string
		for _, asset := range release.Assets {
			if *asset.Name == "source.tar.gz" {
				sourceURL = asset.BrowserDownloadURL
				sourceSHA256 = asset.Digest
				break
			}
		}
		if sourceURL == nil || sourceSHA256 == nil {
			return nil, errors.New("Failed to find source asset")
		}

		for inArch, outArch := range ArchMap {
			assetName := fmt.Sprintf(archAsset, inArch)
			for _, asset := range release.Assets {
				if *asset.Name == assetName {
					result = append(result,
						GitHubRelease{
							version:      version,
							Arch:         outArch,
							BinaryURL:    *asset.BrowserDownloadURL,
							BinarySHA256: *asset.Digest,
							SourceURL:    *sourceURL,
							SourceSHA256: *sourceSHA256,
							UploadTime:   *asset.UpdatedAt.GetTime(),
						})
					break
				}
			}
		}
	}

	return result, nil
}

func getAllUvVersions() (versionology.VersionFetcherArray, error) {
	return getGitHubVersions("astral-sh", "uv", "uv-%s-unknown-linux-gnu.tar.gz")
}

func generateUvMetadata(versionFetcher versionology.VersionFetcher) ([]versionology.Dependency, error) {
	version := versionFetcher.Version().String()
	uvRelease, ok := versionFetcher.(GitHubRelease)
	if !ok {
		return nil, errors.New("expected a GitHubRelease")
	}

	var licenseIDsAsInterface []interface{}
	licenseIDsAsInterface = append(licenseIDsAsInterface, "Apache-2.0", "MIT")
	configMetadataDependency := cargo.ConfigMetadataDependency{
		CPE:            fmt.Sprintf("cpe:2.3:a:uv:uv:%s:*:*:*:*:python:*:*", version),
		Checksum:       uvRelease.BinarySHA256,
		ID:             "uv",
		Licenses:       licenseIDsAsInterface,
		Name:           "uv",
		OS:             "linux",
		Arch:           uvRelease.Arch,
		PURL:           retrieve.GeneratePURL("uv", version, uvRelease.SourceSHA256, uvRelease.SourceURL),
		Source:         uvRelease.SourceURL,
		SourceChecksum: uvRelease.SourceSHA256,
		Stacks:         []string{"*"},
		URI:            uvRelease.BinaryURL,
		Version:        version,
	}

	return []versionology.Dependency{{
		ConfigMetadataDependency: configMetadataDependency,
		SemverVersion:            versionFetcher.Version(),
	}}, nil
}

func getAllPixiVersions() (versionology.VersionFetcherArray, error) {
	return getGitHubVersions("prefix-dev", "pixi", "pixi-%s-unknown-linux-musl.tar.gz")
}

func generatePixiMetadata(versionFetcher versionology.VersionFetcher) ([]versionology.Dependency, error) {
	version := versionFetcher.Version().String()
	pixiRelease, ok := versionFetcher.(GitHubRelease)
	if !ok {
		return nil, errors.New("expected a GitHubRelease")
	}

	fmt.Printf("version: %s\n", version)

	var licenseIDsAsInterface []interface{}
	licenseIDsAsInterface = append(licenseIDsAsInterface, "BSD-3-Clause")
	configMetadataDependency := cargo.ConfigMetadataDependency{
		CPE:            fmt.Sprintf("cpe:2.3:a:pixi:pixi:%s:*:*:*:*:python:*:*", version),
		Checksum:       pixiRelease.BinarySHA256,
		ID:             "pixi",
		Licenses:       licenseIDsAsInterface,
		Name:           "pixi",
		OS:             "linux",
		Arch:           pixiRelease.Arch,
		PURL:           retrieve.GeneratePURL("pixi", version, pixiRelease.SourceSHA256, pixiRelease.SourceURL),
		Source:         pixiRelease.SourceURL,
		SourceChecksum: pixiRelease.SourceSHA256,
		Stacks:         []string{"*"},
		URI:            pixiRelease.BinaryURL,
		Version:        version,
	}

	return []versionology.Dependency{{
		ConfigMetadataDependency: configMetadataDependency,
		SemverVersion:            versionFetcher.Version(),
	}}, nil
}

// Taken from libdependency.retrieve.retrieval
// https://github.com/joshuatcasey/libdependency/blob/main/retrieve/retrieval.go
func toWorkflowJson(item any) (string, error) {
	if bytes, err := json.Marshal(item); err != nil {
		return "", err
	} else {
		return string(bytes), nil
	}
}

func main() {
	buildpackTomlPathUsage := "full path to the buildpack.toml file, using only one of camelCase, snake_case, or dash_case"
	var buildpackTomlPath string
	var output string

	flag.StringVar(&buildpackTomlPath, "buildpack_toml_path", buildpackTomlPath, buildpackTomlPathUsage)
	flag.StringVar(&output, "output", "", "filename for the output JSON metadata")
	flag.Parse()

	exists, err := fs.Exists(buildpackTomlPath)

	if err != nil {
		panic(err)
	} else if !exists {
		panic(fmt.Errorf("could not locate buildpack.toml at '%s'", buildpackTomlPath))
	}

	if output == "" {
		panic("output is required")
	}

	config, err := buildpack_config.ParseBuildpackToml(buildpackTomlPath)
	if err != nil {
		panic(err)
	}

	metadataGeneratorMap := map[string]retrieve.GenerateMetadataFunc{
		"pip":        generatePipMetadata,
		"pipenv":     generatePipenvMetadata,
		"poetry":     generatePoetryMetadata,
		"miniconda3": generateMinicondaMetadata,
		"uv":         generateUvMetadata,
		"pixi":       generatePixiMetadata,
	}

	var dependencies []versionology.Dependency

	for id, generateMetadata := range metadataGeneratorMap {
		newVersions, err := retrieve.GetNewVersionsForId(id, config, getAllVersionsForInstaller(id))
		if err != nil {
			panic(err)
		}

		// This loop is taken from GenerateAllMetadata
		// https://github.com/joshuatcasey/libdependency/blob/main/retrieve/retrieval.go
		for _, version := range newVersions {
			metadata, err := generateMetadata(version)
			if err != nil {
				panic(err)
			}

			var targets []string
			for _, metadatum := range metadata {
				targets = append(targets, metadatum.Target)
			}
			fmt.Printf("Generating metadata for %s, with targets [%s]\n", version.Version().String(), strings.Join(targets, ", "))
			dependencies = append(dependencies, metadata...)
		}
	}

	metadataJson, err := toWorkflowJson(dependencies)
	if err != nil {
		panic(fmt.Errorf("unable to marshall metadata json, with error=%w", err))
	}

	if err = os.WriteFile(output, []byte(metadataJson), os.ModePerm); err != nil {
		panic(fmt.Errorf("cannot write to %s: %w", output, err))
	} else {
		fmt.Printf("Wrote metadata to %s\n", output)
	}
}
