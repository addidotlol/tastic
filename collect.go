package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const collectDirEnv = "TASTIC_COLLECT_DIR"

type externalDiffJob struct {
	Args []string `json:"args"`
}

func (j externalDiffJob) path() string {
	if len(j.Args) == 9 {
		return j.Args[7]
	}
	return j.Args[0]
}

func (j externalDiffJob) size() int64 {
	var total int64
	for _, file := range []string{j.Args[1], j.Args[4]} {
		if info, err := os.Stat(file); err == nil {
			total += info.Size()
		}
	}
	return total
}

func isExternalDiffInvocation() bool {
	return os.Getenv(collectDirEnv) != ""
}

func stageExternalDiff(args []string) error {
	if len(args) != 7 && len(args) != 9 {
		return fmt.Errorf("expected 7 or 9 arguments from git, got %d", len(args))
	}
	root := os.Getenv(collectDirEnv)
	existing, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	jobDir := filepath.Join(root, fmt.Sprintf("job-%06d", len(existing)))
	if err := os.Mkdir(jobDir, 0o700); err != nil {
		return err
	}

	job := externalDiffJob{Args: append([]string{}, args...)}
	newName := filepath.Base(job.path())
	if job.Args[1], err = stageFile(args[1], filepath.Join(jobDir, "old"), filepath.Base(args[0])); err != nil {
		return err
	}
	if job.Args[4], err = stageFile(args[4], filepath.Join(jobDir, "new"), newName); err != nil {
		return err
	}

	encoded, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(jobDir, "job.json"), encoded, 0o600)
}

func stageFile(source, dir, name string) (string, error) {
	if source == os.DevNull {
		return source, nil
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		return "", err
	}
	target := filepath.Join(dir, name)
	if err := os.Link(source, target); err == nil {
		return target, nil
	}
	return target, copyFile(source, target)
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func readStagedJobs(root string) ([]externalDiffJob, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	jobs := make([]externalDiffJob, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "job-") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, entry.Name(), "job.json"))
		if err != nil {
			return nil, err
		}
		var job externalDiffJob
		if err := json.Unmarshal(raw, &job); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
