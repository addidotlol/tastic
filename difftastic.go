package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

type byteSpan [2]int

type difftasticFile struct {
	Path      string             `json:"path"`
	Language  string             `json:"language"`
	Status    string             `json:"status"`
	Deletions map[int][]byteSpan `json:"deletions"`
	Additions map[int][]byteSpan `json:"additions"`
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

func parseDifftastic(r io.Reader) ([]difftasticFile, error) {
	decoder := json.NewDecoder(r)
	var files []difftasticFile
	for {
		var report difftasticReport
		err := decoder.Decode(&report)
		if errors.Is(err, io.EOF) {
			return files, nil
		}
		if err != nil {
			return nil, fmt.Errorf("reading difftastic JSON: %w", err)
		}
		files = append(files, report.toFile())
	}
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
