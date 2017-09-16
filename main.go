package main

import (
	"bytes"
	"context"
	"flag"
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

	buf := new(bytes.Buffer)
	tw := throttle.NewWriter(os.Stdin, buf)

	go tw.Run()

	c := make(chan os.Signal, 0)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

	interval := time.Tick(duration)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{}, 0)

	param := &slack.SlackPostTextParam{
		Channel:   channel,
		Username:  username,
		IconEmoji: ":rocket:",
	}

	go func(ctx context.Context) {
		for {
			select {
			case <-interval:
				break
			case <-ctx.Done():
				tw.Flush()

				param.Text = buf.String()
				buf.Reset()

				err = sClient.PostText(context.Background(), param)
				if err != nil {
					panic(err)
				}

				done <- struct{}{}
				return
			}
			tw.Flush()

			param.Text = buf.String()
			buf.Reset()

			err = sClient.PostText(context.Background(), param)
			if err != nil {
				panic(err)
			}
		}
	}(ctx)

	select {
	case <-c:
	case <-tw.Exit():
	}
	cancel()

	<-done
}
