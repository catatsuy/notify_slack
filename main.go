package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func main() {
	b, _ := json.Marshal(struct {
		Channel   string `json:"channel"`
		Username  string `json:"username"`
		Text      string `json:"text"`
		IconEmoji string `json:"icon_emoji"`
	}{
		Channel:   "#tester",
		Username:  "waiwai",
		Text:      "This is posted to #tester",
		IconEmoji: ":ghost:",
	})

	req, err := http.NewRequest("POST", "https://hooks.slack.com/services/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", bytes.NewBuffer(b))
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
