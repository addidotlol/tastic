package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
)

//go:embed assets/page.html
var pageTemplateSource string

//go:embed assets/viewer.js
var viewerScript string

//go:embed assets/viewer.css
var viewerStyles string

var pageTemplate = template.Must(template.New("page").Parse(pageTemplateSource))

type viewerData struct {
	Repository  string           `json:"repository"`
	Branch      string           `json:"branch"`
	Range       string           `json:"range"`
	GeneratedAt string           `json:"generatedAt"`
	Patch       string           `json:"patch"`
	Files       []difftasticFile `json:"files"`
}

type pageValues struct {
	Title  string
	Styles template.CSS
	Data   template.JS
	Script template.JS
}

func renderPage(data viewerData) ([]byte, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("encoding viewer data: %w", err)
	}

	var page bytes.Buffer
	err = pageTemplate.Execute(&page, pageValues{
		Title:  fmt.Sprintf("%s · %s", data.Repository, data.Branch),
		Styles: template.CSS(escapeClosingTag(viewerStyles, "</style")),
		Data:   template.JS(payload),
		Script: template.JS(escapeClosingTag(viewerScript, "</script")),
	})
	if err != nil {
		return nil, fmt.Errorf("rendering page: %w", err)
	}
	return page.Bytes(), nil
}

func escapeClosingTag(source, tag string) string {
	return strings.ReplaceAll(source, tag, `<\/`+tag[2:])
}
