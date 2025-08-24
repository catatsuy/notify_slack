package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/catatsuy/notify_slack/internal/config"
	"github.com/catatsuy/notify_slack/internal/slack"
	"github.com/catatsuy/notify_slack/internal/throttle"
)

var (
	Version string
)

const (
	ExitCodeOK             = 0
	ExitCodeParseFlagError = 1
	ExitCodeFail           = 1
)

type CLI struct {
	outStream, errStream io.Writer
	inputStream          io.Reader

	isStdinTerminal bool

	sClient    slack.Slack
	conf       *config.Config
	appVersion string
}

func NewCLI(outStream, errStream io.Writer, inputStream io.Reader, isStdinTerminal bool) *CLI {
	return &CLI{appVersion: version(), outStream: outStream, errStream: errStream, inputStream: inputStream, isStdinTerminal: isStdinTerminal}
}

func version() string {
	if Version != "" {
		return Version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(devel)"
	}
	return info.Main.Version
}

type cliOptions struct {
	version        bool
	tomlFile       string
	uploadFilename string
	filetype       string
	snippetMode    bool
	debugMode      bool
	filename       string
}

func (c *CLI) Run(args []string) int {
	opts, err := c.parseFlags(args)
	if err != nil {
		return ExitCodeParseFlagError
	}

	if opts.version {
		c.printVersion()
		return ExitCodeOK
	}

	if err := c.loadConfiguration(opts.tomlFile); err != nil {
		fmt.Fprintln(c.errStream, err)
		return ExitCodeFail
	}

	logger := c.createLogger(opts.debugMode)
	ctx := context.Background()

	if opts.filename != "" || opts.snippetMode {
		return c.handleSnippetMode(ctx, opts, logger)
	}

	return c.handleTextMode(ctx, logger)
}

func (c *CLI) parseFlags(args []string) (*cliOptions, error) {
	opts := &cliOptions{}
	c.conf = config.NewConfig()

	flags := flag.NewFlagSet("notify_slack", flag.ContinueOnError)
	flags.SetOutput(c.errStream)

	// Set up all flags
	c.setupFlags(flags, opts)

	if err := flags.Parse(args[1:]); err != nil {
		return nil, err
	}

	// Check version flag first - it doesn't need file arguments
	if opts.version {
		return opts, nil
	}

	// Process remaining arguments
	argv := flags.Args()
	if err := c.processArguments(argv, opts, flags); err != nil {
		return nil, err
	}

	return opts, nil
}

func (c *CLI) setupFlags(flags *flag.FlagSet, opts *cliOptions) {
	flags.StringVar(&c.conf.Channel, "channel", "", "specify channel (unavailable for new Incoming Webhooks)")
	flags.StringVar(&c.conf.ChannelID, "channel-id", "", "specify channel id (for uploading a file)")
	flags.StringVar(&c.conf.SlackURL, "slack-url", "", "slack url (Incoming Webhooks URL)")
	flags.StringVar(&c.conf.Token, "token", "", "token (for uploading to snippet)")
	flags.StringVar(&c.conf.Username, "username", "", "specify username (unavailable for new Incoming Webhooks)")
	flags.StringVar(&c.conf.IconEmoji, "icon-emoji", "", "specify icon emoji (unavailable for new Incoming Webhooks)")
	flags.DurationVar(&c.conf.Duration, "interval", time.Second, "interval")
	flags.StringVar(&opts.tomlFile, "c", "", "config file name")
	flags.StringVar(&opts.uploadFilename, "filename", "", "specify a file name (for uploading to snippet)")
	flags.StringVar(&opts.filetype, "filetype", "", "[compatible] specify a filetype for uploading to snippet. This option is maintained for compatibility. Please use -snippet-type instead.")
	flags.StringVar(&opts.filetype, "snippet-type", "", "specify a snippet_type (for uploading to snippet)")
	flags.BoolVar(&opts.snippetMode, "snippet", false, "switch to snippet uploading mode")
	flags.BoolVar(&opts.debugMode, "debug", false, "debug mode (for developers)")
	flags.BoolVar(&opts.version, "version", false, "Print version information and quit")
}

func (c *CLI) processArguments(argv []string, opts *cliOptions, flags *flag.FlagSet) error {
	if len(argv) == 1 {
		opts.filename = argv[0]
	} else if len(argv) > 1 {
		opts.filename = argv[0]
		if err := flags.Parse(argv[1:]); err != nil {
			return err
		}
		argv = flags.Args()
		if len(argv) > 0 {
			fmt.Fprintln(c.errStream, "You cannot pass multiple files")
			return fmt.Errorf("multiple files specified")
		}
	} else if c.isStdinTerminal {
		fmt.Fprintln(c.errStream, "No input file specified")
		return fmt.Errorf("no input file specified")
	}
	return nil
}

func (c *CLI) printVersion() {
	fmt.Fprintf(c.errStream, "notify_slack version %s; %s\n", c.appVersion, runtime.Version())
}

func (c *CLI) loadConfiguration(tomlFile string) error {
	tomlFile = config.LoadTOMLFilename(tomlFile)
	if tomlFile != "" {
		if err := c.conf.LoadTOML(tomlFile); err != nil {
			return err
		}
	}
	return c.conf.LoadEnv()
}

func (c *CLI) createLogger(debugMode bool) *slog.Logger {
	if debugMode {
		return slog.New(slog.NewTextHandler(c.errStream, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}))
	}
	return slog.New(slog.NewTextHandler(c.errStream, nil))
}

func (c *CLI) handleSnippetMode(ctx context.Context, opts *cliOptions, logger *slog.Logger) int {
	if c.conf.Token == "" {
		fmt.Fprintln(c.errStream, "must specify Slack token for uploading to snippet")
		return ExitCodeFail
	}

	var err error
	c.sClient, err = slack.NewClientForPostFile(c.conf.Token, logger)
	if err != nil {
		fmt.Fprintln(c.errStream, err)
		return ExitCodeFail
	}

	if err := c.uploadSnippet(ctx, opts.filename, opts.uploadFilename, opts.filetype); err != nil {
		fmt.Fprintln(c.errStream, err)
		return ExitCodeFail
	}

	return ExitCodeOK
}

func (c *CLI) handleTextMode(ctx context.Context, logger *slog.Logger) int {
	if c.conf.SlackURL == "" {
		fmt.Fprintln(c.errStream, "must specify Slack URL")
		return ExitCodeFail
	}

	var err error
	c.sClient, err = slack.NewClient(c.conf.SlackURL, logger)
	if err != nil {
		fmt.Fprintln(c.errStream, err)
		return ExitCodeFail
	}

	return c.streamToSlack(ctx)
}

func (c *CLI) streamToSlack(ctx context.Context) int {
	copyStdin := io.TeeReader(c.inputStream, c.outStream)
	ex := throttle.NewExec(copyStdin)

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	param := &slack.PostTextParam{
		Channel:   c.conf.Channel,
		Username:  c.conf.Username,
		IconEmoji: c.conf.IconEmoji,
	}

	flushCallback := func(ctx context.Context, output string) error {
		param.Text = output
		return c.sClient.PostText(context.WithoutCancel(ctx), param)
	}

	done := make(chan struct{})
	doneCallback := func(ctx context.Context, output string) error {
		defer func() {
			go func() {
				done <- struct{}{}
			}()
		}()
		return flushCallback(context.WithoutCancel(ctx), output)
	}

	ticker := time.NewTicker(c.conf.Duration)
	defer ticker.Stop()

	ex.Start(ctx, ticker.C, flushCallback, doneCallback)
	<-done

	return ExitCodeOK
}

func (c *CLI) uploadSnippet(ctx context.Context, filename, uploadFilename, snippetType string) error {
	channelID := c.conf.ChannelID

	var reader io.ReadCloser
	if filename == "" {
		reader = os.Stdin
	} else {
		_, err := os.Stat(filename)
		if err != nil {
			return fmt.Errorf("%s does not exist: %w", filename, err)
		}
		reader, err = os.Open(filename)
		if err != nil {
			return fmt.Errorf("can't open %s: %w", filename, err)
		}
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	if uploadFilename == "" {
		uploadFilename = filename
	}

	param := &slack.PostFileParam{
		ChannelID:   channelID,
		Filename:    uploadFilename,
		SnippetType: snippetType,
	}

	err = c.sClient.PostFile(ctx, param, content)
	if err != nil {
		return err
	}

	return nil
}
