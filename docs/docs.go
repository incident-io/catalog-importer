package docs

import (
	"embed"
	"path"
	"strings"

	"github.com/pkg/errors"
)

//go:embed */*
var Content embed.FS

type Template struct {
	Name        string // e.g. backstage
	Label       string // e.g. Backstage
	Description string // e.g. This template is ...
}

func GetTemplates() ([]Template, error) {
	entries, err := Content.ReadDir(".")
	if err != nil {
		return nil, errors.Wrap(err, "reading doc directory")
	}

	templates := []Template{}
	for _, entry := range entries {
		data, err := Content.ReadFile(path.Join(entry.Name(), "README.md"))
		if err != nil {
			return nil, errors.Wrap(err, "reading README.md")
		}

		var preamble string
		preamble = strings.SplitN(string(data), "Out the box", 2)[0] // keep everything above
		preamble = strings.SplitN(preamble, "\n\n", 2)[1]            // drop the header

		templates = append(templates, Template{
			Name:        entry.Name(),
			Label:       strings.TrimPrefix(strings.Split(string(data), "\n")[0], "# "),
			Description: preamble,
		})
	}

	return templates, nil
}
