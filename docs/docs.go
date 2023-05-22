package docs

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/google/go-jsonnet"
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

// EvaluateJsonnet returns the JSON document produced when evaluating a Jsonnet file
// within the docs directory.
func EvaluateJsonnet(wd, file string) ([]byte, error) {
	importer := jsonnet.MemoryImporter{
		Data: map[string]jsonnet.Contents{},
	}
	err := fs.WalkDir(Content, wd, fs.WalkDirFunc(func(entryPath string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		src, err := Content.Open(entryPath)
		if err != nil {
			return err
		}

		data, err := io.ReadAll(src)
		if err != nil {
			return err
		}

		importer.Data[strings.TrimPrefix(entryPath, fmt.Sprintf("%s/", wd))] =
			jsonnet.MakeContentsRaw(data)

		return nil
	}))
	if err != nil {
		return nil, err
	}

	vm := jsonnet.MakeVM()
	vm.Importer(&importer)

	data, err := vm.EvaluateFile(file)
	if err != nil {
		return nil, err
	}

	return []byte(data), nil
}
