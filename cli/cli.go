package cli

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/catatsuy/notify_slack/config"
	"github.com/catatsuy/notify_slack/slack"
	"github.com/catatsuy/notify_slack/throttle"
)

const (
	ExitCodeOK             = 0
	ExitCodeParseFlagError = 1
)

type CLI struct {
	outStream, errStream io.Writer
	inputStream          io.Reader
}

func NewCLI(outStream, errStream io.Writer, inputStream io.Reader) *CLI {
	return &CLI{outStream: outStream, errStream: errStream, inputStream: inputStream}
}

func (c *CLI) Run(args []string) int {
	var (
		tomlFile string
		duration time.Duration
	)

	conf := config.NewConfig()

	flags := flag.NewFlagSet("notify_slack", flag.ContinueOnError)
	flags.SetOutput(c.errStream)

	flags.StringVar(&conf.Channel, "channel", "", "specify channel")
	flags.StringVar(&conf.SlackURL, "slack-url", "", "slack url")
	flags.StringVar(&conf.Token, "token", "", "token")
	flags.StringVar(&conf.Username, "username", "", "specify username")
	flags.StringVar(&conf.IconEmoji, "icon-emoji", "", "specify icon emoji")

	flags.DurationVar(&duration, "interval", time.Second, "interval")
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
		conf.LoadTOML(tomlFile)
	}

	if conf.SlackURL == "" {
		log.Fatal("provide Slack URL")
	}

	sClient, err := slack.NewClient(conf.SlackURL, nil)
	if err != nil {
		log.Fatal(err)
	}

	if filename != "" {
		if conf.Token == "" {
			log.Fatal("provide Slack token")
		}

		_, err = os.Stat(filename)
		if err != nil {
			log.Fatalf("%s does not exist", filename)
		}

		content, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}

		param := &slack.PostFileParam{
			Channel:  conf.Channel,
			Filename: filename,
			Content:  string(content),
		}
		err = sClient.PostFile(context.Background(), conf.Token, param)
		if err != nil {
			log.Fatal(err)
		}

		return ExitCodeOK
	}

	copyStdin := io.TeeReader(c.inputStream, c.outStream)

	ex := throttle.NewExec(copyStdin)

	exitC := make(chan os.Signal, 0)
	signal.Notify(exitC, syscall.SIGTERM, syscall.SIGINT)

	param := &slack.PostTextParam{
		Channel:   conf.Channel,
		Username:  conf.Username,
		IconEmoji: conf.IconEmoji,
	}

	flushCallback := func(_ context.Context, output string) error {
		param.Text = output
		return sClient.PostText(context.Background(), param)
	}

	done := make(chan struct{}, 0)

	doneCallback := func(ctx context.Context, output string) error {
		defer func() {
			done <- struct{}{}
		}()

		return flushCallback(ctx, output)
	}

	interval := time.Tick(duration)
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
