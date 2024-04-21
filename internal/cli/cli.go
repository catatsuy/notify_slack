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
	"golang.org/x/term"
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

	sClient    slack.Slack
	conf       *config.Config
	appVersion string
}

func NewCLI(outStream, errStream io.Writer, inputStream io.Reader) *CLI {
	return &CLI{appVersion: version(), outStream: outStream, errStream: errStream, inputStream: inputStream}
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

func (c *CLI) Run(args []string) int {
	var (
		version        bool
		tomlFile       string
		uploadFilename string
		filetype       string
		snippetMode    bool
		debugMode      bool
	)

	c.conf = config.NewConfig()

	flags := flag.NewFlagSet("notify_slack", flag.ContinueOnError)
	flags.SetOutput(c.errStream)

	flags.StringVar(&c.conf.Channel, "channel", "", "specify channel (unavailable for new Incoming Webhooks)")
	flags.StringVar(&c.conf.ChannelID, "channel-id", "", "specify channel id (for uploading a file)")
	flags.StringVar(&c.conf.SlackURL, "slack-url", "", "slack url (Incoming Webhooks URL)")
	flags.StringVar(&c.conf.Token, "token", "", "token (for uploading a file)")
	flags.StringVar(&c.conf.Username, "username", "", "specify username (unavailable for new Incoming Webhooks)")
	flags.StringVar(&c.conf.IconEmoji, "icon-emoji", "", "specify icon emoji (unavailable for new Incoming Webhooks)")
	flags.DurationVar(&c.conf.Duration, "interval", time.Second, "interval")
	flags.StringVar(&tomlFile, "c", "", "config file name")
	flags.StringVar(&uploadFilename, "filename", "", "specify a file name (for uploading a file)")
	flags.StringVar(&filetype, "filetype", "", "specify a filetype (for uploading a file)")

	flags.BoolVar(&snippetMode, "snippet", false, "switch to file uploading mode")

	flags.BoolVar(&debugMode, "debug", false, "debug mode (for developers)")

	flags.BoolVar(&version, "version", false, "Print version information and quit")

	err := flags.Parse(args[1:])
	if err != nil {
		return ExitCodeParseFlagError
	}

	if version {
		fmt.Fprintf(c.errStream, "notify_slack version %s; %s\n", c.appVersion, runtime.Version())
		return ExitCodeOK
	}

	argv := flags.Args()
	filename := ""
	if len(argv) == 1 {
		filename = argv[0]
	} else if len(argv) > 1 {
		filename = argv[0]
		err = flags.Parse(argv[1:])
		if err != nil {
			return ExitCodeParseFlagError
		}

		argv = flags.Args()
		if len(argv) > 0 {
			fmt.Fprintln(c.errStream, "You cannot pass multiple files")
			return ExitCodeParseFlagError
		}
	} else if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintln(c.errStream, "No input file specified")
		return ExitCodeFail
	}

	tomlFile = config.LoadTOMLFilename(tomlFile)

	if tomlFile != "" {
		err := c.conf.LoadTOML(tomlFile)
		if err != nil {
			fmt.Fprintln(c.errStream, err)
			return ExitCodeFail
		}
	}

	err = c.conf.LoadEnv()
	if err != nil {
		fmt.Fprintln(c.errStream, err)
		return ExitCodeFail
	}

	var logger *slog.Logger
	if debugMode {
		logger = slog.New(slog.NewTextHandler(c.errStream, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}))
	} else {
		logger = slog.New(slog.NewTextHandler(c.errStream, nil))
	}

	if filename != "" || snippetMode {
		if c.conf.Token == "" {
			fmt.Fprintln(c.errStream, "must specify Slack token for uploading to snippet")
			return ExitCodeFail
		}

		c.sClient, err = slack.NewClientForFile(c.conf.Token, logger)
		if err != nil {
			fmt.Fprintln(c.errStream, err)
			return ExitCodeFail
		}

		err := c.uploadSnippet(context.Background(), filename, uploadFilename, filetype)
		if err != nil {
			fmt.Fprintln(c.errStream, err)
			return ExitCodeFail
		}

		return ExitCodeOK
	}

	if c.conf.SlackURL == "" {
		fmt.Fprintln(c.errStream, "must specify Slack URL")
		return ExitCodeFail
	}

	c.sClient, err = slack.NewClient(c.conf.SlackURL, logger)
	if err != nil {
		fmt.Fprintln(c.errStream, err)
		return ExitCodeFail
	}

	copyStdin := io.TeeReader(c.inputStream, c.outStream)

	ex := throttle.NewExec(copyStdin)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	channel := c.conf.Channel

	param := &slack.PostTextParam{
		Channel:   channel,
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
			// If goroutine is not used, it will not exit when the pipe is closed
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

	params := &slack.PostFileParam{
		ChannelID:   channelID,
		Filename:    uploadFilename,
		SnippetType: snippetType,
	}

	err = c.sClient.PostFile(ctx, params, content)
	if err != nil {
		return err
	}

	return nil
}
