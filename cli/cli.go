package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/catatsuy/notify_slack/config"
	"github.com/catatsuy/notify_slack/slack"
	"github.com/catatsuy/notify_slack/throttle"
	"github.com/pkg/errors"
)

const (
	ExitCodeOK             = 0
	ExitCodeParseFlagError = 1
	ExitCodeFail           = 1
)

type CLI struct {
	outStream, errStream io.Writer
	inputStream          io.Reader

	sClient *slack.Client
	conf    *config.Config
}

func NewCLI(outStream, errStream io.Writer, inputStream io.Reader) *CLI {
	return &CLI{outStream: outStream, errStream: errStream, inputStream: inputStream}
}

func (c *CLI) Run(args []string) int {
	var (
		tomlFile string
	)

	c.conf = config.NewConfig()

	flags := flag.NewFlagSet("notify_slack", flag.ContinueOnError)
	flags.SetOutput(c.errStream)

	flags.StringVar(&c.conf.Channel, "channel", "", "specify channel")
	flags.StringVar(&c.conf.SlackURL, "slack-url", "", "slack url")
	flags.StringVar(&c.conf.Token, "token", "", "token")
	flags.StringVar(&c.conf.Username, "username", "", "specify username")
	flags.StringVar(&c.conf.IconEmoji, "icon-emoji", "", "specify icon emoji")

	flags.DurationVar(&c.conf.Duration, "interval", time.Second, "interval")
	flags.StringVar(&tomlFile, "c", "", "config file name")

	err := flags.Parse(args[1:])
	if err != nil {
		return ExitCodeParseFlagError
	}

	argv := flags.Args()
	filename := ""
	if len(argv) == 1 {
		filename = argv[0]
	}

	tomlFile = config.LoadTOMLFilename(tomlFile)

	if tomlFile != "" {
		err := c.conf.LoadTOML(tomlFile)
		if err != nil {
			fmt.Fprintln(c.errStream, err)
			return ExitCodeFail
		}
	}

	if c.conf.SlackURL == "" {
		fmt.Fprintln(c.errStream, "provide Slack URL")
		return ExitCodeFail
	}

	c.sClient, err = slack.NewClient(c.conf.SlackURL, nil)
	if err != nil {
		fmt.Fprintln(c.errStream, err)
		return ExitCodeFail
	}

	if filename != "" {
		if c.conf.Token == "" {
			fmt.Fprintln(c.errStream, "provide Slack token")
			return ExitCodeFail
		}

		err := c.uploadSnippet(context.Background(), filename)
		if err != nil {
			fmt.Fprintln(c.errStream, err)
			return ExitCodeFail
		}

		return ExitCodeOK
	}

	copyStdin := io.TeeReader(c.inputStream, c.outStream)

	ex := throttle.NewExec(copyStdin)

	exitC := make(chan os.Signal, 0)
	signal.Notify(exitC, syscall.SIGTERM, syscall.SIGINT)

	param := &slack.PostTextParam{
		Channel:   c.conf.Channel,
		Username:  c.conf.Username,
		IconEmoji: c.conf.IconEmoji,
	}

	flushCallback := func(_ context.Context, output string) error {
		param.Text = output
		return c.sClient.PostText(context.Background(), param)
	}

	done := make(chan struct{}, 0)

	doneCallback := func(ctx context.Context, output string) error {
		defer func() {
			done <- struct{}{}
		}()

		return flushCallback(ctx, output)
	}

	interval := time.Tick(c.conf.Duration)
	ctx, cancel := context.WithCancel(context.Background())

	ex.Start(ctx, interval, flushCallback, doneCallback)

	select {
	case <-exitC:
	case <-ex.Wait():
	}
	cancel()

	<-done

	return ExitCodeOK
}

func (c *CLI) uploadSnippet(ctx context.Context, filename string) error {
	_, err := os.Stat(filename)
	if err != nil {
		return errors.Wrapf(err, "%s does not exist", filename)
	}

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	param := &slack.PostFileParam{
		Channel:  c.conf.Channel,
		Filename: filename,
		Content:  string(content),
	}
	err = c.sClient.PostFile(ctx, c.conf.Token, param)
	if err != nil {
		return err
	}

	return nil
}
