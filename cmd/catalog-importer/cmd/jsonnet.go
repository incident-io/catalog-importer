package cmd

import (
	"context"
	"fmt"

	"github.com/alecthomas/kingpin/v2"
	kitlog "github.com/go-kit/kit/log"
	"github.com/google/go-jsonnet"
)

type JsonnetOptions struct {
	Filename string
}

func (opt *JsonnetOptions) Bind(cmd *kingpin.CmdClause) *JsonnetOptions {
	cmd.Arg("file", "File containing Jsonnet").
		StringVar(&opt.Filename)

	return opt
}

func (opt *JsonnetOptions) Run(ctx context.Context, logger kitlog.Logger) error {
	data, err := jsonnet.MakeVM().EvaluateFile(opt.Filename)
	if err != nil {
		return err
	}

	fmt.Println(data)
	return nil
}
