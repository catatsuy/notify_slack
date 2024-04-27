package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

var (
	slackFilesUploadURL = "https://slack.com/api/files.upload"
)

type Client struct {
	Slack

	URL        *url.URL
	HTTPClient *http.Client

	Token string

	Logger *slog.Logger
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
	Filetype string
}

type Slack interface {
	PostText(ctx context.Context, param *PostTextParam) error
	PostFile(ctx context.Context, param *PostFileParam) error
}

func NewClient(urlStr string, logger *slog.Logger) (*Client, error) {
	if len(urlStr) == 0 {
		return nil, fmt.Errorf("client: missing url")
	}

	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %s: %w", urlStr, err)
	}

	client := &Client{
		URL:        parsedURL,
		HTTPClient: http.DefaultClient,
		Logger:     logger,
	}

	return client, nil
}

func NewClientForPostFile(token string, logger *slog.Logger) (*Client, error) {
	if len(token) == 0 {
		return nil, fmt.Errorf("provide Slack token")
	}

	client := &Client{
		HTTPClient: http.DefaultClient,
		Token:      token,
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

	c.Logger.Debug("request", "url", req.URL.String(), "method", req.Method, "header", req.Header)

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read res.Body: %w", err)
	}

	c.Logger.Debug("request", "url", req.URL.String(), "method", req.Method, "header", req.Header, "status", res.StatusCode, "body", body)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d; body: %s", res.StatusCode, body)
	}

	return nil
}

type apiFilesUploadRes struct {
	OK bool `json:"ok"`
}

func (c *Client) PostFile(ctx context.Context, param *PostFileParam) error {
	if param.Content == "" {
		return fmt.Errorf("the content of the file is empty")
	}

	v := url.Values{}
	v.Set("token", c.Token)
	v.Set("content", param.Content)
	v.Set("filename", param.Filename)
	v.Set("channels", param.Channel)

	if param.Filetype != "" {
		v.Set("filetype", param.Filetype)
	}

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
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	c.Logger.Debug("request", "url", req.URL.String(), "method", req.Method, "header", req.Header, "status", res.StatusCode, "body", b)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to read res.Body and the status code of the response from slack was not 200; body: %s", b)
	}

	apiRes := apiFilesUploadRes{}
	err = json.Unmarshal(b, &apiRes)
	if err != nil {
		return fmt.Errorf("response returned from slack is not json: %w", err)
	}

	if !apiRes.OK {
		return fmt.Errorf("response has failed; body: %s", b)
	}
	return nil
}
