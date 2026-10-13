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

func (r *repository) difftastic(difft string, diffArgs []string) ([]difftasticFile, error) {
	env := []string{
		"GIT_EXTERNAL_DIFF=" + difft,
		"DFT_DISPLAY=json",
		"DFT_UNSTABLE=yes",
		"DFT_COLOR=never",
	}
	args := append([]string{"diff", "--ext-diff", "--no-color", "-M"}, diffArgs...)
	out, err := r.git(env, args...)
	if err != nil {
		return nil, fmt.Errorf("git diff through difftastic: %w", err)
	}
	return parseDifftastic(strings.NewReader(out))
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
