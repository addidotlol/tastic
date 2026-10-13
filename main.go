package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type options struct {
	output      string
	noUpload    bool
	description string
	difft       string
	graphLimit  int
	diffArgs    []string
}

func main() {
	if isExternalDiffInvocation() {
		if err := stageExternalDiff(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "tastic: staging file pair for difftastic:", err)
			os.Exit(1)
		}
		return
	}
	if err := run(parseOptions()); err != nil {
		fmt.Fprintln(os.Stderr, "tastic:", err)
		os.Exit(1)
	}
}

func parseOptions() options {
	var opts options
	flag.StringVar(&opts.output, "o", "", "write the HTML here (default: a file in the system temp dir)")
	flag.BoolVar(&opts.noUpload, "no-upload", false, "only write the HTML file, skip postplan.dev")
	flag.StringVar(&opts.description, "description", "", "description shown on the postplan.dev dashboard")
	flag.StringVar(&opts.difft, "difft", "difft", "difftastic binary to use")
	flag.IntVar(&opts.graphLimit, "graph-limit", 1_000_000, "DFT_GRAPH_LIMIT for difftastic; lower is faster, 0 keeps difftastic's default")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "usage: tastic [flags] [-- git diff args]")
		fmt.Fprintln(flag.CommandLine.Output(), "\nRenders the current repo's git diff through difftastic and publishes it to postplan.dev.")
		fmt.Fprintln(flag.CommandLine.Output(), "Without git diff args, tastic diffs the working tree against HEAD.")
		fmt.Fprintln(flag.CommandLine.Output())
		flag.PrintDefaults()
	}
	flag.Parse()
	opts.diffArgs = flag.Args()
	return opts
}

func run(opts options) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("reading current directory: %w", err)
	}

	repo, err := openRepository(cwd)
	if err != nil {
		return err
	}

	difft, err := findDifftastic(opts.difft)
	if err != nil {
		return err
	}

	diffArgs := opts.diffArgs
	if len(diffArgs) == 0 {
		diffArgs = repo.defaultDiffArgs()
	}

	patch, err := repo.patch(diffArgs)
	if err != nil {
		return err
	}
	if patch == "" {
		return errors.New("no changes to show")
	}

	files, err := repo.difftastic(difftasticRunner{binary: difft, graphLimit: opts.graphLimit}, diffArgs)
	if err != nil {
		return err
	}

	page, err := renderPage(viewerData{
		Repository:  repo.name(),
		Branch:      repo.branch(),
		Range:       joinArgs(diffArgs),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Patch:       patch,
		Files:       files,
	})
	if err != nil {
		return err
	}

	output := opts.output
	if output == "" {
		output = defaultOutputPath(repo.name())
	}
	if err := writeFile(output, page); err != nil {
		return err
	}
	fmt.Println("wrote", output)

	if opts.noUpload {
		return nil
	}

	receipt, err := uploadToPostplan(page, uploadDetails{
		filename:    filepath.Base(output),
		description: opts.description,
		repoName:    repo.name(),
		branch:      repo.branch(),
	})
	if err != nil {
		return err
	}
	fmt.Println(receipt.PublicURL)
	return nil
}

func defaultOutputPath(repoName string) string {
	stamp := time.Now().Format("20060102-150405")
	return filepath.Join(os.TempDir(), "tastic", fmt.Sprintf("%s-%s.html", repoName, stamp))
}

func writeFile(path string, contents []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
