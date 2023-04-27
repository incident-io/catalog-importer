package cmd

import (
	"context"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	_ "embed"

	"github.com/alecthomas/kingpin/v2"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
)

var logger kitlog.Logger

var (
	app = kingpin.New("catalog-importer", "Import data into your incident.io catalog").Version(versionStanza())

	// Global flags
	debug = app.Flag("debug", "Enable debug logging").Default("false").Bool()

	// Sync
	sync        = app.Command("sync", "Sync data from catalog sources into incident.io")
	syncOptions = new(SyncOptions).Bind(sync)

	// Docs
	docs        = app.Command("docs", "Need help? Run this for links to docs and an example config reference")
	docsOptions = new(DocsOptions).Bind(docs)

	// Validate
	validate        = app.Command("validate", "Validate configuration")
	validateOptions = new(ValidateOptions).Bind(validate)
)

func Run(ctx context.Context) (err error) {
	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))
	if *debug {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}
	logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC, "caller", kitlog.DefaultCaller)
	logger = level.Debug(logger) // by default, logger is debug only
	stdlog.SetOutput(kitlog.NewStdlibAdapter(logger))

	// Root context to the application.
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	go func() {
		<-sigc
		cancel()
		<-sigc
		panic("received second signal, exiting immediately")
	}()

	switch command {
	case sync.FullCommand():
		return syncOptions.Run(ctx, logger)
	case docs.FullCommand():
		return docsOptions.Run(ctx, logger)
	case validate.FullCommand():
		return validateOptions.Run(ctx, logger)
	default:
		return fmt.Errorf("unrecognised command: %s", command)
	}
}

// Set via compiler flags
var (
	Commit    = "none"
	Date      = "unknown"
	GoVersion = runtime.Version()
)

//go:embed VERSION
var Version string

func versionStanza() string {
	return fmt.Sprintf(
		"Version: %v\nGit SHA: %v\nGo Version: %v\nGo OS/Arch: %v/%v\nBuilt at: %v",
		Version, Commit, GoVersion, runtime.GOOS, runtime.GOARCH, Date,
	)
}

// OUT prints progress output to stderr.
func OUT(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
}

func BANNER(msg string, args ...any) {
	msg = strings.Join(
		[]string{
			"################################################################################",
			"# " + msg,
			"################################################################################",
		},
		"\n",
	)

	OUT(msg, args...)
}
