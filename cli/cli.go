package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/catatsuy/notify_slack/config"
	"github.com/catatsuy/notify_slack/slack"
	"github.com/catatsuy/notify_slack/throttle"
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
	)

	c.conf = config.NewConfig()

	flags := flag.NewFlagSet("notify_slack", flag.ContinueOnError)
	flags.SetOutput(c.errStream)

	flags.StringVar(&c.conf.PrimaryChannel, "channel", "", "specify channel (unavailable for new Incoming Webhooks)")
	flags.StringVar(&c.conf.SlackURL, "slack-url", "", "slack url (Incoming Webhooks URL)")
	flags.StringVar(&c.conf.Token, "token", "", "token (for uploading to snippet)")
	flags.StringVar(&c.conf.Username, "username", "", "specify username (unavailable for new Incoming Webhooks)")
	flags.StringVar(&c.conf.IconEmoji, "icon-emoji", "", "specify icon emoji (unavailable for new Incoming Webhooks)")
	flags.DurationVar(&c.conf.Duration, "interval", time.Second, "interval")
	flags.StringVar(&tomlFile, "c", "", "config file name")
	flags.StringVar(&uploadFilename, "filename", "", "specify a file name (for uploading to snippet)")
	flags.StringVar(&filetype, "filetype", "", "specify a filetype (for uploading to snippet)")

	flags.BoolVar(&snippetMode, "snippet", false, "switch to snippet uploading mode")

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
	}

	tomlFile = config.LoadTOMLFilename(tomlFile)

	if tomlFile != "" {
		err := c.conf.LoadTOML(tomlFile)
		if err != nil {
			fmt.Fprintln(c.errStream, err)
			return ExitCodeFail
		}
	}

	// environment variables
	if c.conf.SlackURL == "" {
		c.conf.SlackURL = os.Getenv("NOTIFY_SLACK_WEBHOOK_URL")
	}
	if c.conf.Token == "" {
		c.conf.Token = os.Getenv("NOTIFY_SLACK_TOKEN")
	}
	if c.conf.Channel == "" {
		c.conf.Channel = os.Getenv("NOTIFY_SLACK_CHANNEL")
	}
	if c.conf.SnippetChannel == "" {
		c.conf.SnippetChannel = os.Getenv("NOTIFY_SLACK_SNIPPET_CHANNEL")
	}
	if c.conf.Username == "" {
		c.conf.Username = os.Getenv("NOTIFY_SLACK_USERNAME")
	}
	if c.conf.IconEmoji == "" {
		c.conf.IconEmoji = os.Getenv("NOTIFY_SLACK_ICON_EMOJI")
	}

	if filename != "" || snippetMode {
		c.sClient, err = slack.NewClientForPostFile(nil)
		if err != nil {
			fmt.Fprintln(c.errStream, err)
			return ExitCodeFail
		}

		if c.conf.Token == "" {
			fmt.Fprintln(c.errStream, "must specify Slack token for uploading to snippet")
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

	c.sClient, err = slack.NewClient(c.conf.SlackURL, nil)
	if err != nil {
		fmt.Fprintln(c.errStream, err)
		return ExitCodeFail
	}

	copyStdin := io.TeeReader(c.inputStream, c.outStream)

	ex := throttle.NewExec(copyStdin)

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)

	channel := c.conf.PrimaryChannel
	if channel == "" {
		channel = c.conf.Channel
	}

	param := &slack.PostTextParam{
		Channel:   channel,
		Username:  c.conf.Username,
		IconEmoji: c.conf.IconEmoji,
	}

	flushCallback := func(output string) error {
		param.Text = output
		return c.sClient.PostText(context.Background(), param)
	}

	done := make(chan struct{})

	doneCallback := func(output string) error {
		defer func() {
			// If goroutine is not used, it will not exit when the pipe is closed
			go func() {
				done <- struct{}{}
			}()
		}()

		return flushCallback(output)
	}

	ticker := time.NewTicker(c.conf.Duration)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	exitC := make(chan struct{})
	go func() {
		ex.Start(ctx, ticker.C, flushCallback, doneCallback)
		close(exitC)
	}()

	select {
	case <-sigC:
	case <-exitC:
	}
	cancel()

	<-done

	return ExitCodeOK
}

func (c *CLI) uploadSnippet(ctx context.Context, filename, uploadFilename, filetype string) error {
	channel := c.conf.PrimaryChannel
	if channel == "" {
		channel = c.conf.SnippetChannel
	}
	if channel == "" {
		channel = c.conf.Channel
	}

	if channel == "" {
		return fmt.Errorf("must specify channel for uploading to snippet")
	}

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

	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	if uploadFilename == "" {
		uploadFilename = filename
	}

	param := &slack.PostFileParam{
		Channel:  channel,
		Filename: uploadFilename,
		Content:  string(content),
		Filetype: filetype,
	}
	err = c.sClient.PostFile(ctx, c.conf.Token, param)
	if err != nil {
		return err
	}

	return nil
}
