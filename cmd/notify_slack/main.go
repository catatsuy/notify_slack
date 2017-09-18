package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/catatsuy/notify_slack/config"
	"github.com/catatsuy/notify_slack/slack"
	"github.com/catatsuy/notify_slack/throttle"
)

func main() {
	var (
		tomlFile string
		duration time.Duration
	)

	conf := config.NewConfig()

	flag.StringVar(&conf.Channel, "channel", "", "specify channel")
	flag.StringVar(&conf.SlackURL, "slack-url", "", "slack url")
	flag.StringVar(&conf.Username, "username", "", "specify username")
	flag.StringVar(&conf.IconEmoji, "icon-emoji", "", "specify icon emoji")

	flag.DurationVar(&duration, "interval", time.Second, "interval")
	flag.StringVar(&tomlFile, "f", "", "config file name")

	flag.Parse()

	tomlFile = config.LoadTOMLFilename(tomlFile)

	if tomlFile != "" {
		conf.LoadTOML(tomlFile)
	}

	if conf.SlackURL == "" {
		log.Fatal("provide Slack URL")
	}

	sClient, err := slack.NewClient(conf.SlackURL, nil)
	if err != nil {
		panic(err)
	}

	copyStdin := io.TeeReader(os.Stdin, os.Stdout)

	ex := throttle.NewExec(copyStdin)

	c := make(chan os.Signal, 0)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

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
	case <-c:
	case <-ex.Wait():
	}
	cancel()

	<-done
}
