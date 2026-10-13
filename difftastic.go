package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"
)

const difftasticTimeout = 30 * time.Second

type byteSpan [2]int

type difftasticFile struct {
	Path      string             `json:"path"`
	Language  string             `json:"language"`
	Status    string             `json:"status"`
	Deletions map[int][]byteSpan `json:"deletions"`
	Additions map[int][]byteSpan `json:"additions"`
	Failure   string             `json:"failure,omitempty"`
}

type difftasticReport struct {
	Path     string             `json:"path"`
	Language string             `json:"language"`
	Status   string             `json:"status"`
	Chunks   [][]difftasticPair `json:"chunks"`
}

type difftasticPair struct {
	LHS *difftasticLine `json:"lhs"`
	RHS *difftasticLine `json:"rhs"`
}

type difftasticLine struct {
	LineNumber int                `json:"line_number"`
	Changes    []difftasticChange `json:"changes"`
}

type difftasticChange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

func findDifftastic(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("difftastic (%s) not found: install it from https://difftastic.wilfred.me.uk or pass -difft", name)
	}
	return path, nil
}

type difftasticRunner struct {
	binary     string
	graphLimit int
}

func (d difftasticRunner) runAll(jobs []externalDiffJob) []difftasticFile {
	files := make([]difftasticFile, len(jobs))
	progress := newPawgress("difftastic", len(jobs))
	defer progress.finish()

	indexes := make(chan int)
	var workers sync.WaitGroup
	for range min(runtime.NumCPU(), max(len(jobs), 1)) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range indexes {
				files[index] = d.run(jobs[index])
				progress.step(jobs[index].path())
			}
		}()
	}
	for _, index := range largestFirst(jobs) {
		indexes <- index
	}
	close(indexes)
	workers.Wait()
	return files
}

func largestFirst(jobs []externalDiffJob) []int {
	sizes := make([]int64, len(jobs))
	order := make([]int, len(jobs))
	for index, job := range jobs {
		order[index] = index
		sizes[index] = job.size()
	}
	sort.SliceStable(order, func(a, b int) bool { return sizes[order[a]] > sizes[order[b]] })
	return order
}

func (d difftasticRunner) environment() []string {
	env := append(os.Environ(), "DFT_DISPLAY=json", "DFT_UNSTABLE=yes", "DFT_COLOR=never")
	if d.graphLimit > 0 {
		env = append(env, "DFT_GRAPH_LIMIT="+strconv.Itoa(d.graphLimit))
	}
	return env
}

func (d difftasticRunner) run(job externalDiffJob) difftasticFile {
	ctx, cancel := context.WithTimeout(context.Background(), difftasticTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, d.binary, job.Args...)
	cmd.Env = d.environment()
	out, err := cmd.Output()

	var report difftasticReport
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return failedFile(job, "timed out")
	case err != nil:
		return failedFile(job, "crashed")
	case json.Unmarshal(out, &report) != nil:
		return failedFile(job, "returned unreadable output")
	}
	file := report.toFile()
	file.Path = job.path()
	return file
}

func failedFile(job externalDiffJob, reason string) difftasticFile {
	return difftasticFile{Path: job.path(), Status: "failed", Failure: reason}
}

func (r difftasticReport) toFile() difftasticFile {
	file := difftasticFile{
		Path:      r.Path,
		Language:  r.Language,
		Status:    r.Status,
		Deletions: map[int][]byteSpan{},
		Additions: map[int][]byteSpan{},
	}
	for _, chunk := range r.Chunks {
		for _, pair := range chunk {
			addSpans(file.Deletions, pair.LHS)
			addSpans(file.Additions, pair.RHS)
		}
	}
	return file
}

func addSpans(byLine map[int][]byteSpan, line *difftasticLine) {
	if line == nil || len(line.Changes) == 0 {
		return
	}
	oneBasedLine := line.LineNumber + 1
	for _, change := range line.Changes {
		byLine[oneBasedLine] = append(byLine[oneBasedLine], byteSpan{change.Start, change.End})
	}
}
