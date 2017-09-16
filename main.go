package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
)

func main() {
	var (
		channel  string
		username string
		slackURL string
	)

	flag.StringVar(&channel, "channel", "", "specify channel")
	flag.StringVar(&slackURL, "slack-url", "", "slack url")
	flag.StringVar(&username, "username", "", "specify username")
	flag.Parse()

	if slackURL == "" {
		return
	}

	b, _ := json.Marshal(struct {
		Channel   string `json:"channel,omitempty"`
		Username  string `json:"username,omitempty"`
		Text      string `json:"text"`
		IconEmoji string `json:"icon_emoji,omitempty"`
	}{
		Channel:   channel,
		Username:  username,
		Text:      "This is posted to #tester",
		IconEmoji: ":rocket:",
	})

	req, err := http.NewRequest("POST", slackURL, bytes.NewBuffer(b))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(body))
}
