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

	"github.com/pkg/errors"
)

type Client struct {
	URL        *url.URL
	HTTPClient *http.Client

	Logger *log.Logger
}

type SlackPostTextParam struct {
	Channel   string `json:"channel,omitempty"`
	Username  string `json:"username,omitempty"`
	Text      string `json:"text"`
	IconEmoji string `json:"icon_emoji,omitempty"`
}

func NewClient(urlStr string, logger *log.Logger) (*Client, error) {
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

	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (c *Client) PostText(ctx context.Context, param *SlackPostTextParam) error {
	b, _ := json.Marshal(param)

	req, err := c.newRequest(ctx, "POST", bytes.NewBuffer(b))
	if err != nil {
		return err
	}

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
