package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type repository struct {
	cwd  string
	root string
}

func openRepository(cwd string) (*repository, error) {
	repo := &repository{cwd: cwd}
	root, err := repo.git(nil, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("%s is not inside a git work tree", cwd)
	}
	repo.root = strings.TrimSpace(root)
	return repo, nil
}

func (r *repository) name() string {
	return filepath.Base(r.root)
}

func (r *repository) branch() string {
	if branch, err := r.git(nil, "symbolic-ref", "--short", "-q", "HEAD"); err == nil {
		return strings.TrimSpace(branch)
	}
	if commit, err := r.git(nil, "rev-parse", "--short", "HEAD"); err == nil {
		return strings.TrimSpace(commit)
	}
	return "unknown"
}

func (r *repository) hasCommits() bool {
	_, err := r.git(nil, "rev-parse", "--verify", "-q", "HEAD")
	return err == nil
}

func (r *repository) defaultDiffArgs() []string {
	if r.hasCommits() {
		return []string{"HEAD"}
	}
	return nil
}

func (r *repository) patch(diffArgs []string) (string, error) {
	args := append([]string{"diff", "--no-ext-diff", "--no-color", "-M"}, diffArgs...)
	out, err := r.git(nil, args...)
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return out, nil
}

func (r *repository) difftastic(runner difftasticRunner, diffArgs []string) ([]difftasticFile, error) {
	jobs, cleanup, err := r.stageFilePairs(diffArgs)
	defer cleanup()
	if err != nil {
		return nil, err
	}
	return runner.runAll(jobs), nil
}

func (r *repository) stageFilePairs(diffArgs []string) ([]externalDiffJob, func(), error) {
	noop := func() {}
	self, err := os.Executable()
	if err != nil {
		return nil, noop, fmt.Errorf("locating the tastic binary: %w", err)
	}
	stagingDir, err := os.MkdirTemp("", "tastic-pairs-")
	if err != nil {
		return nil, noop, err
	}
	cleanup := func() { os.RemoveAll(stagingDir) }

	env := []string{
		"GIT_EXTERNAL_DIFF=" + shellQuote(self),
		collectDirEnv + "=" + stagingDir,
	}
	args := append([]string{"diff", "--ext-diff", "--no-color", "-M"}, diffArgs...)
	if _, err := r.git(env, args...); err != nil {
		return nil, cleanup, fmt.Errorf("collecting changed files: %w", err)
	}
	jobs, err := readStagedJobs(stagingDir)
	return jobs, cleanup, err
}

func (r *repository) git(extraEnv []string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-c", "core.quotepath=false"}, args...)...)
	cmd.Dir = r.cwd
	cmd.Env = append(os.Environ(), extraEnv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return "", err
		}
		return "", errors.New(message)
	}
	return stdout.String(), nil
}

func joinArgs(args []string) string {
	return strings.Join(args, " ")
}
