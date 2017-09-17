package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/catatsuy/notify_slack/slack"
	"github.com/catatsuy/notify_slack/throttle"
)

func main() {
	var (
		channel  string
		username string
		slackURL string
		text     string
		duration time.Duration
	)

	flag.StringVar(&channel, "channel", "", "specify channel")
	flag.StringVar(&slackURL, "slack-url", "", "slack url")
	flag.StringVar(&username, "username", "", "specify username")
	flag.StringVar(&text, "text", "", "text")
	flag.DurationVar(&duration, "interval", time.Second, "interval")

	flag.Parse()

	if slackURL == "" {
		return
	}

	sClient, err := slack.NewClient(slackURL, nil)
	if err != nil {
		panic(err)
	}

	copyStdin := io.TeeReader(os.Stdin, os.Stdout)

	buf := new(bytes.Buffer)
	tw := throttle.NewWriter(copyStdin, buf)

	c := make(chan os.Signal, 0)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

	param := &slack.SlackPostTextParam{
		Channel:   channel,
		Username:  username,
		IconEmoji: ":rocket:",
	}

	flushCallback := func(ctx context.Context, output string) error {
		param.Text = output
		return sClient.PostText(context.Background(), param)
	}

	done := make(chan struct{}, 0)

	doneCallback := func(ctx context.Context, output string) error {
		err := flushCallback(ctx, output)

		if err != nil {
			return err
		}

		done <- struct{}{}

		return nil
	}

	interval := time.Tick(duration)
	ctx, cancel := context.WithCancel(context.Background())

	tw.Setup()

	select {
	case <-c:
	case <-tw.Run(ctx, interval, flushCallback, doneCallback):
	}
	cancel()

	<-done
}
