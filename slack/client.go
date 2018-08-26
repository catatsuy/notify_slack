package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

var (
	slackFilesUploadURL = "https://slack.com/api/files.upload"
)

type Client struct {
	URL        *url.URL
	Token      string
	HTTPClient *http.Client

	Logger *log.Logger
}

type PostTextParam struct {
	Channel   string `json:"channel,omitempty"`
	Username  string `json:"username,omitempty"`
	Text      string `json:"text"`
	IconEmoji string `json:"icon_emoji,omitempty"`
}

type PostFileParam struct {
	Channel  string
	Content  string
	Filename string
}

func NewClient(urlStr string, token string, logger *log.Logger) (*Client, error) {
	if len(urlStr) == 0 {
		return nil, fmt.Errorf("client: missing url")
	}

	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse url: %s", urlStr)
	}

	var discardLogger = log.New(ioutil.Discard, "", log.LstdFlags)
	if logger == nil {
		logger = discardLogger
	}

	client := &Client{
		URL:        parsedURL,
		Token:      token,
		HTTPClient: http.DefaultClient,
		Logger:     logger,
	}

	return client, nil
}

func (c *Client) newRequest(ctx context.Context, method string, body io.Reader) (*http.Request, error) {
	u := *c.URL

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	return req, nil
}

func (c *Client) PostText(ctx context.Context, param *PostTextParam) error {
	if param.Text == "" {
		return nil
	}

	b, _ := json.Marshal(param)

	req, err := c.newRequest(ctx, "POST", bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return errors.Wrap(err, "Failed to read res.Body and the status code of the response from slack was not 200")
		}
		return fmt.Errorf("status code: %d; body: %s", res.StatusCode, b)
	}

	return nil
}

type apiFilesUploadRes struct {
	OK bool `json:"ok"`
}

func (c *Client) PostFile(ctx context.Context, param *PostFileParam) error {
	v := url.Values{}
	v.Set("token", c.Token)
	v.Set("content", param.Content)
	v.Set("filename", param.Filename)
	v.Set("channels", param.Channel)

	req, err := http.NewRequest("POST", slackFilesUploadURL, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to read res.Body and the status code of the response from slack was not 200; body: %s", b)
	}

	apiRes := apiFilesUploadRes{}
	err = json.Unmarshal(b, &apiRes)
	if err != nil {
		return errors.Wrap(err, "response returned from slack is not json")
	}

	if !apiRes.OK {
		return fmt.Errorf("response has failed; body: %s", b)
	}
	return nil
}
