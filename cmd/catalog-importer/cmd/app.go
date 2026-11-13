package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	_ "embed"

	"github.com/alecthomas/kingpin/v2"
	"github.com/fatih/color"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/go-cmp/cmp"
	"github.com/incident-io/catalog-importer/v2/config"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

var logger kitlog.Logger

// outputLogger writes progress output as JSON lines when --log-format is json, so the
// output can be parsed alongside the logs. It's nil otherwise, and progress output is
// written to the terminal as plain text.
var outputLogger kitlog.Logger

var (
	app = kingpin.New("catalog-importer", "Import data into your incident.io catalog").Version(versionStanza())

	// Global flags
	debug     = app.Flag("debug", "Enable debug logging").Default("false").Bool()
	quiet     = app.Flag("quiet", "Suppress all non-error output").Default("false").Bool()
	noColor   = app.Flag("no-color", "Disable colored output").Default("false").Bool()
	logFormat = app.Flag("log-format", "Format in which to emit logs (either json or logfmt). json also writes progress output as JSON lines, without colors or progress bars").Default("logfmt").String()

	// Init
	initCmd     = app.Command("init", "Initialises a new config from a template")
	initOptions = new(InitOptions).Bind(initCmd)

	// import
	importCmd     = app.Command("import", "Import catalog data directly or generate importer config")
	importOptions = new(ImportOptions).Bind(importCmd)

	// Types
	typesCmd     = app.Command("types", "Shows all the types that can be used for this account")
	typesOptions = new(TypesOptions).Bind(typesCmd)

	// Sync
	sync        = app.Command("sync", "Sync data from catalog sources into incident.io")
	syncOptions = new(SyncOptions).Bind(sync)

	// Source
	sourceCmd     = app.Command("source", "Loads and prints the catalog entries from source, for debugging")
	sourceOptions = new(SourceOptions).Bind(sourceCmd)

	// Jsonnet
	jsonnetCmd     = app.Command("jsonnet", "Evaluate Jsonnet files")
	jsonnetOptions = new(JsonnetOptions).Bind(jsonnetCmd)

	// Validate
	validate        = app.Command("validate", "Validate configuration")
	validateOptions = new(ValidateOptions).Bind(validate)

	// Backstage
	backstageCmd     = app.Command("backstage", "Syncs catalog entries directly from Backstage API into incident.io")
	backstageOptions = new(BackstageOptions).Bind(backstageCmd)
)

func Run(ctx context.Context) (err error) {
	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	switch *logFormat {
	case "json":
		logger = kitlog.NewJSONLogger(kitlog.NewSyncWriter(os.Stderr))
		outputLogger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)
		color.NoColor = true
	case "logfmt":
		logger = kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))
	default:
		return fmt.Errorf("invalid log format %q: must be 'json' or 'logfmt'", *logFormat)
	}
	if *debug {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}
	logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC, "caller", kitlog.DefaultCaller)
	logger = level.Debug(logger) // by default, logger is debug only
	stdlog.SetOutput(kitlog.NewStdlibAdapter(logger))

	// Root context to the application.
	ctx, cancel := context.WithCancel(context.Background())
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
	case initCmd.FullCommand():
		return initOptions.Run(ctx, logger)
	case importCmd.FullCommand():
		return importOptions.Run(ctx, logger)
	case typesCmd.FullCommand():
		return typesOptions.Run(ctx, logger)
	case sync.FullCommand():
		return syncOptions.Run(ctx, logger, nil)
	case sourceCmd.FullCommand():
		return sourceOptions.Run(ctx, logger)
	case jsonnetCmd.FullCommand():
		return jsonnetOptions.Run(ctx, logger)
	case validate.FullCommand():
		return validateOptions.Run(ctx, logger)
	case backstageCmd.FullCommand():
		return backstageOptions.Run(ctx, logger)
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
var version string

func Version() string {
	return strings.TrimSpace(version)
}

func versionStanza() string {
	return fmt.Sprintf(
		"Version: %v\nGit SHA: %v\nGo Version: %v\nGo OS/Arch: %v/%v\nBuilt at: %v",
		Version(), Commit, GoVersion, runtime.GOOS, runtime.GOARCH, Date,
	)
}

func loadConfigOrError(ctx context.Context, configFile string) (cfg *config.Config, err error) {
	defer func() {
		if err == nil {
			return
		}
		if configFile == "" {
			ALWAYS_OUT("No config file (--config) was provided, but is required.\n")
		} else {
			ALWAYS_OUT("Failed to load config file! It's either in an unexpected format, or we can't find it using that path.\n")
		}

		// The highlighted example is written straight to stdout, so it's only shown in a
		// terminal.
		if outputLogger == nil {
			ALWAYS_OUT(`We expect a config file in Jsonnet, JSON or YAML format that looks like:
`)
			config.PrettyPrint(`// e.g. importer.jsonnet
{
  sync_id: 'unique-sync-id',
  pipelines: [
    {
      sources: [/* where to load catalog data */],
      outputs: [/* which catalog types to push data into, and how */],
    }
  ],
}`)
		}

		ALWAYS_OUT(`
View reference config file in GitHub: https://github.com/incident-io/catalog-importer/blob/master/config/reference.jsonnet
`)
	}()

	if configFile == "" {
		return nil, errors.New("No config file set! (--config)")
	}

	cfg, err = config.FileLoader(configFile).Load(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "loading config")
	}

	if err := cfg.Validate(); err != nil {
		data, _ := json.MarshalIndent(err, "", "  ")

		// Print the validation error in JSON. Needs improving.
		return nil, fmt.Errorf("validating config:\n%s", string(data))
	}

	return cfg, nil
}

// OUT prints progress output to stderr, unless `--quiet` is set.
func OUT(msg string, args ...any) {
	if lo.FromPtr(quiet) {
		return
	}

	ALWAYS_OUT(msg, args...)
}

// ALWAYS_OUT prints output to stderr, even if --quiet is set.
func ALWAYS_OUT(msg string, args ...any) {
	if outputLogger != nil {
		logOutput(level.Info(outputLogger), fmt.Sprintf(msg, args...))
		return
	}

	fmt.Fprintf(os.Stderr, msg+"\n", args...)
}

// logOutput writes progress output as a log entry, without the blank lines and
// indentation that lay it out in a terminal.
func logOutput(logger kitlog.Logger, msg string, keyvals ...any) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}

	logger.Log(append([]any{"msg", msg}, keyvals...)...)
}

// exit is replaced in tests, so Fatal can be called without ending the test run.
var exit = os.Exit

// Fatal reports the error that stopped the command, and exits.
func Fatal(err error) {
	if outputLogger != nil {
		logOutput(level.Error(outputLogger), err.Error())
	} else {
		app.Errorf("%s", err.Error())
	}

	exit(1)
}

func BANNER(msg string, args ...any) {
	if outputLogger != nil {
		ALWAYS_OUT(msg, args...)
		return
	}

	msg = strings.Join(
		[]string{
			"################################################################################",
			"# " + msg,
			"################################################################################",
		},
		"\n",
	)

	ALWAYS_OUT(msg, args...)
}

func DIFF[Type any](prefix string, this, that Type) {
	thisJSON, _ := json.Marshal(this)
	var thisNormalised any
	json.Unmarshal(thisJSON, &thisNormalised)

	thatJSON, _ := json.Marshal(that)
	var thatNormalised any
	json.Unmarshal(thatJSON, &thatNormalised)

	var buf strings.Builder

	diff := cmp.Diff(thisNormalised, thatNormalised)
	if outputLogger != nil {
		if diff != "" && !lo.FromPtr(quiet) {
			logOutput(level.Info(outputLogger), "diff", "diff", diff)
		}
		return
	}

	if diff != "" {
		// Set up colored output
		green := color.New(color.FgGreen)
		red := color.New(color.FgRed)

		// Disable colors if --no-color flag is set
		if *noColor {
			color.NoColor = true
		}

		for _, line := range strings.Split(diff, "\n") {
			// Add the prefix, such as constant whitespace.
			buf.WriteString(prefix)

			if strings.HasPrefix(line, "+") {
				buf.WriteString(green.Sprint(line) + "\n")
			} else if strings.HasPrefix(line, "-") {
				buf.WriteString(red.Sprint(line) + "\n")
			} else {
				buf.WriteString(line + "\n")
			}
		}
	}

	if strings.TrimSpace(buf.String()) != "" {
		OUT("%s", strings.TrimRight(buf.String(), "\n "))
	}
}
