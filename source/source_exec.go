package source

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	kitlog "github.com/go-kit/kit/log"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/pkg/errors"
)

type SourceExec struct {
	Command []string `json:"command"`
}

func (s SourceExec) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Command, validation.Length(1, 0).
			Error("must provide at least a command, if no args")),
	)
}

func (s SourceExec) String() string {
	return fmt.Sprintf("exec (command=%s)", strings.Join(s.Command, ","))
}

func (s SourceExec) Load(ctx context.Context, logger kitlog.Logger) ([]*SourceEntry, error) {
	var (
		command = s.Command[0]
		args    = s.Command[1:]
	)
	cmd := exec.CommandContext(ctx, command, args...)

	// If the exec'd command goes wrong, then we want to be able to see both the
	// stdout and stderr in the terminal, but if it's succeeded then we need to
	// have captured stdout, to construct the entry.
	var output bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &output)
	cmd.Stderr = os.Stderr // allow stderr output

	err := cmd.Run()
	if err != nil {
		return nil, errors.Wrap(err, "error running exec command")
	}

	entries := []*SourceEntry{
		{
			Origin:  fmt.Sprintf("exec: %s", strings.Join(s.Command, " ")),
			Content: output.Bytes(),
		},
	}

	return entries, nil
}
