package assets

import (
	"embed"
	"text/template"
)

//go:embed templates/*
var Templates embed.FS

func GetTemplate(name string) (*template.Template, error) {
	b, err := Templates.ReadFile("templates/" + name)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(name).Parse(string(b))
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}
