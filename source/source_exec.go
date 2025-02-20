package source

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
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

func (s SourceExec) Load(ctx context.Context, logger kitlog.Logger, _ *http.Client) ([]*SourceEntry, error) {
	var (
		command = s.Command[0]
		args    = s.Command[1:]
	)
	cmd := exec.CommandContext(ctx, command, args...)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = os.Stderr // stderr is streamed to the parent terminal

	err := cmd.Run()
	if err != nil {
		// If the exec'd command fails, then it's sometimes useful to see the standard output
		// that it produced. This is especially relevant when the program doesn't behave
		// according to conventions, and the error message isn't actually present in stderr.
		// In some cases though, the stream of data could be very large, so we deliberately
		// only show the last 1KiB, to avoid filling up the logs/terminal.
		// We write this output back onto stderr, to play nice with any downstream tooling.
		limit := 1024
		stdout := output.Bytes()

		if len(stdout) > limit {
			fmt.Fprintln(os.Stderr, "last 1KiB of failing command's stdout:")

			stdout = stdout[len(stdout)-limit:]
		} else if len(stdout) > 0 {
			fmt.Fprintln(os.Stderr, "failing command's stdout:")
		}

		fmt.Fprintln(os.Stderr, string(stdout))

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
