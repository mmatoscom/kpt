// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/kptfile"
	"github.com/stretchr/testify/assert"
	assertnow "gotest.tools/assert"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const TmpDirPrefix = "test-kpt"

const (
	Dataset1            = "dataset1"
	Dataset2            = "dataset2"
	Dataset3            = "dataset3"
	Dataset4            = "dataset4" // Dataset4 is replica of Dataset2 with different setter values
	DatasetMerged       = "datasetmerged"
	DiffOutput          = "diff_output"
	UpdateMergeConflict = "updateMergeConflict"
	HelloWorldSet       = "helloworld-set"
)

// TestGitRepo manages a local git repository for testing
type TestGitRepo struct {
	// RepoDirectory is the temp directory of the git repo
	RepoDirectory string

	// DatasetDirectory is the directory of the testdata files
	DatasetDirectory string

	// RepoName is the name of the repository
	RepoName string

	Updater string
}

var AssertNoError = assertnow.NilError

var KptfileSet = func() sets.String {
	s := sets.String{}
	s.Insert(kptfile.KptFileName)
	return s
}()

// AssertEqual verifies the contents of a source package matches the contents of the
// destination package it was fetched to.
// Excludes comparing the .git directory in the source package.
//
// While the sourceDir can be the TestGitRepo, because tests change the TestGitRepo
// may have been changed after the destDir was copied, it is often better to explicitly
// use a set of golden files as the sourceDir rather than the original TestGitRepo
// that was copied.
func (g *TestGitRepo) AssertEqual(t *testing.T, sourceDir, destDir string) bool {
	diff, err := copyutil.Diff(sourceDir, destDir)
	if !assert.NoError(t, err) {
		return false
	}
	diff = diff.Difference(KptfileSet)
	return assert.Empty(t, diff.List(), g.Updater)
}

func AssertEqual(t *testing.T, g *TestGitRepo, sourceDir, destDir string) {
	diff, err := copyutil.Diff(sourceDir, destDir)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	diff = diff.Difference(KptfileSet)
	if !assert.Empty(t, diff.List(), g.Updater) {
		t.FailNow()
	}
}

func Replace(t *testing.T, path, old, new string) {
	b, err := ioutil.ReadFile(path)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// update the expected contents to reflect the set command
	b = []byte(strings.Replace(
		string(b), old, new, -1))
	if !assert.NoError(t, ioutil.WriteFile(path, b, 0)) {
		t.FailNow()
	}
}

func Compare(t *testing.T, a, b string) {
	// Compare parses the yaml and serializes both files to normalize
	// formatting
	b1, err := ioutil.ReadFile(a)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	n1, err := yaml.Parse(string(b1))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	s1, err := n1.String()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	b2, err := ioutil.ReadFile(b)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	n2, err := yaml.Parse(string(b2))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	s2, err := n2.String()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Equal(t, s1, s2) {
		t.FailNow()
	}
}

// AssertKptfile verifies the contents of the KptFile matches the provided value.
func (g *TestGitRepo) AssertKptfile(t *testing.T, cloned string, kpkg kptfile.KptFile) bool {
	// read the actual generated KptFile
	b, err := ioutil.ReadFile(filepath.Join(cloned, kptfile.KptFileName))
	if !assert.NoError(t, err, g.Updater) {
		return false
	}
	actual := kptfile.KptFile{}
	d := yaml.NewDecoder(bytes.NewBuffer(b))
	d.KnownFields(true)
	if !assert.NoError(t, d.Decode(&actual), g.Updater) {
		return false
	}
	return assert.Equal(t, kpkg, actual, g.Updater)
}

// CheckoutBranch checks out the git branch in the repo
func (g *TestGitRepo) CheckoutBranch(branch string, create bool) error {
	var args []string
	if create {
		args = []string{"checkout", "-b", branch}
	} else {
		args = []string{"checkout", branch}
	}

	// checkout the branch
	cmd := exec.Command("git", args...)
	cmd.Dir = g.RepoDirectory
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", stdoutStderr)
		return err
	}

	return nil
}

// Commit performs a git commit
func (g *TestGitRepo) Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = g.RepoDirectory
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", stdoutStderr)
		return err
	}
	return nil
}

func Commit(t *testing.T, g *TestGitRepo, message string) {
	if !assert.NoError(t, g.Commit(message)) {
		t.FailNow()
	}
}

func CommitTag(t *testing.T, g *TestGitRepo, tag string) {
	Commit(t, g, tag)
	Tag(t, g, tag)
}

// CopyAddData copies data from a source and adds it
func (g *TestGitRepo) CopyAddData(data string) error {
	if !filepath.IsAbs(data) {
		data = filepath.Join(g.DatasetDirectory, data)
	}

	err := copyutil.CopyDir(data, g.RepoDirectory)
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = g.RepoDirectory
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", stdoutStderr)
		return err
	}

	return nil
}

func CopyData(t *testing.T, g *TestGitRepo, data, dest string) {
	if !filepath.IsAbs(data) {
		data = filepath.Join(g.DatasetDirectory, data)
	}

	dest = filepath.Join(g.RepoDirectory, dest)
	err := os.MkdirAll(dest, 0700)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.NoError(t, copyutil.CopyDir(data, dest)) {
		t.FailNow()
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = g.RepoDirectory
	stdoutStderr, err := cmd.CombinedOutput()
	if !assert.NoError(t, err, stdoutStderr) {
		t.FailNow()
	}
}

func (g *TestGitRepo) GetCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = g.RepoDirectory
	b, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// RemoveAll deletes the test git repo
func (g *TestGitRepo) RemoveAll() error {
	err := os.RemoveAll(g.RepoDirectory)
	return err
}

func RemoveData(t *testing.T, g *TestGitRepo) {
	// remove the old data
	files, err := ioutil.ReadDir(g.RepoDirectory)
	if err != nil {
		t.FailNow()
	}
	for i := range files {
		f := files[i]
		if f.IsDir() && f.Name() == ".git" {
			continue
		}
		err := os.RemoveAll(filepath.Join(g.RepoDirectory, f.Name()))
		if err != nil {
			t.FailNow()
		}
	}
}

// ReplaceData replaces the data with a new source
func (g *TestGitRepo) ReplaceData(data string) error {
	if !filepath.IsAbs(data) {
		data = filepath.Join(g.DatasetDirectory, data)
	}

	// remove the old data
	files, err := ioutil.ReadDir(g.RepoDirectory)
	if err != nil {
		return err
	}
	for i := range files {
		f := files[i]
		if f.IsDir() && f.Name() == ".git" {
			continue
		}
		err := os.RemoveAll(filepath.Join(g.RepoDirectory, f.Name()))
		if err != nil {
			return err
		}
	}

	// copy the new data
	return g.CopyAddData(data)
}

// SetupTestGitRepo initializes a new git repository and populates it with data from a source
func (g *TestGitRepo) SetupTestGitRepo(data string) error {
	// configure the path to the test dataset
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return errors.Errorf("failed to testutil package location")
	}
	ds, err := filepath.Abs(filepath.Join(filepath.Dir(filename), "testdata"))
	if err != nil {
		return err
	}
	g.DatasetDirectory = ds

	// create the test repo directory
	dir, err := ioutil.TempDir("", fmt.Sprintf("%s-upstream-", TmpDirPrefix))
	if err != nil {
		return err
	}
	g.RepoDirectory = dir
	g.RepoName = filepath.Base(g.RepoDirectory)

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", stdoutStderr)
		return err
	}

	// populate the repo with
	err = g.CopyAddData(data)
	if err != nil {
		return err
	}
	return g.Commit("initial commit")
}

// Tag initializes tags the git repository
func (g *TestGitRepo) Tag(tag string) error {
	cmd := exec.Command("git", "tag", tag)
	cmd.Dir = g.RepoDirectory
	b, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", b)
		return err
	}

	return nil
}

func Tag(t *testing.T, g *TestGitRepo, tag string) {
	if !assert.NoError(t, g.Tag(tag)) {
		t.FailNow()
	}
}

func CopyKptfile(t *testing.T, src, dest string) {
	b, err := ioutil.ReadFile(filepath.Join(src, kptfile.KptFileName))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	err = ioutil.WriteFile(filepath.Join(dest, kptfile.KptFileName), b, 0)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
}

// SetupDefaultRepoAndWorkspace handles setting up a default repo to clone, and a workspace to clone into.
// returns a cleanup function to remove the git repo and workspace.
func SetupDefaultRepoAndWorkspace(t *testing.T) (*TestGitRepo, string, func()) {
	// setup the repo to clone from
	g := &TestGitRepo{}
	err := g.SetupTestGitRepo(Dataset1)
	assert.NoError(t, err)

	// setup the directory to clone to
	dir, err := ioutil.TempDir("", "test-kpt-local-")
	assert.NoError(t, err)
	err = os.Chdir(dir)
	assert.NoError(t, err)

	gr := gitutil.NewLocalGitRunner("./")
	if !assert.NoError(t, gr.Run("init")) {
		assert.FailNowf(t, "%s %s", gr.Stdout.String(), gr.Stderr.String())
	}

	return g, dir, func() {
		// ignore cleanup failures
		_ = g.RemoveAll()
		_ = os.RemoveAll(dir)
	}
}
