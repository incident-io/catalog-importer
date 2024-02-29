package cmd

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"

	"github.com/alecthomas/kingpin/v2"
	kitlog "github.com/go-kit/kit/log"
	"github.com/incident-io/catalog-importer/docs"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
)

type InitOptions struct {
}

func (opt *InitOptions) Bind(cmd *kingpin.CmdClause) *InitOptions {
	return opt
}

func (opt *InitOptions) Run(ctx context.Context, logger kitlog.Logger) error {
	configTemplates, err := docs.GetTemplates()
	if err != nil {
		return err
	}

	var chosenTemplate docs.Template
	{
		prompt := promptui.Select{
			Label: "Where do you store your catalog data?",
			Items: configTemplates,
			Size:  4,
			Templates: &promptui.SelectTemplates{
				Label:    "  {{ .Label }}?",
				Active:   "▸ {{ .Label }}",
				Inactive: "  {{ .Label }}",
				Details: `
{{ .Label | bold }}

{{ .Description | faint }}
`,
			},
		}

		idx, _, err := prompt.Run()
		if err != nil {
			return err
		}

		chosenTemplate = configTemplates[idx]
	}

	var chosenDest string
	{
		prompt := promptui.Prompt{
			Label:   "Where should the template be installed?",
			Default: ".",
		}

		chosenDest, err = prompt.Run()
		if err != nil {
			return err
		}
	}

	{
		_, err := os.Stat(chosenDest)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				OUT("destination does not exist, creating...")
				if err := os.MkdirAll(chosenDest, 0755); err != nil {
					return errors.Wrap(err, "creating destination directory")
				}
			} else {
				return errors.Wrap(err, "checking destination directory")
			}
		}
	}

	entries, err := os.ReadDir(chosenDest)
	if err != nil {
		return errors.Wrap(err, "listing destination directory")
	}
	if len(entries) > 0 {
		prompt := promptui.Select{
			Label: "The chosen destination already contains files, are you sure you want to continue?",
			Items: []string{
				"Yes",
				"No",
			},
		}

		_, answer, _ := prompt.Run()
		if answer != "Yes" {
			return nil
		}

	}

	err = fs.WalkDir(docs.Content, chosenTemplate.Name, fs.WalkDirFunc(func(entryPath string, d fs.DirEntry, err error) error {
		destPath := path.Join(chosenDest, entryPath)
		OUT("writing %s...", destPath)

		if _, err := os.Stat(destPath); err == nil {
			return fmt.Errorf("%s already exists, refusing to overwrite", destPath)
		}

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		src, err := docs.Content.Open(entryPath)
		if err != nil {
			return err
		}
		defer src.Close()

		data, err := io.ReadAll(src)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(destPath, data, 0644)
		if err != nil {
			return err
		}

		return nil
	}))
	if err != nil {
		return err
	}

	OUT("\nYour template has been installed at:\n  %s\n", path.Join(chosenDest, chosenTemplate.Name))
	OUT("View the README.md for instructions on how to use it.")

	return nil
}
