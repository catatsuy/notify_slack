package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var (
	filesGetUploadURLExternalURL   = "https://slack.com/api/files.getUploadURLExternal"
	filesCompleteUploadExternalURL = "https://slack.com/api/files.completeUploadExternal"
)

type Client struct {
	Slack

	URL        *url.URL
	HTTPClient *http.Client

	Logger *log.Logger
}

type PostFileParam struct {
	Channel  string
	Content  string
	Filename string
	Filetype string
}

type GetUploadURLExternalRes struct {
	OK        bool   `json:"ok"`
	UploadURL string `json:"upload_url"`
	FileID    string `json:"file_id"`
}

type PostTextParam struct {
	Channel   string `json:"channel,omitempty"`
	Username  string `json:"username,omitempty"`
	Text      string `json:"text"`
	IconEmoji string `json:"icon_emoji,omitempty"`
}

type GetUploadURLExternalResParam struct {
	Filename    string
	SnippetType string
	Length      int
}

type Slack interface {
	PostText(ctx context.Context, param *PostTextParam) error
	PostFile(ctx context.Context, token string, param *PostFileParam) error
}

func NewClient(urlStr string, logger *log.Logger) (*Client, error) {
	if len(urlStr) == 0 {
		return nil, fmt.Errorf("client: missing url")
	}

	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %s: %w", urlStr, err)
	}

	var discardLogger = log.New(io.Discard, "", log.LstdFlags)
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

func NewClientForFile(logger *log.Logger) (*Client, error) {
	var discardLogger = log.New(io.Discard, "", log.LstdFlags)
	if logger == nil {
		logger = discardLogger
	}

	client := &Client{
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
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read res.Body and the status code of the response from slack was not 200: %w", err)
		}
		return fmt.Errorf("status code: %d; body: %s", res.StatusCode, b)
	}

	return nil
}

func NewClientForPostFile(logger *log.Logger) (*Client, error) {
	return nil, nil
}

func (c *Client) GetUploadURLExternalURL(ctx context.Context, token string, param *GetUploadURLExternalResParam) (uploadURL string, fileID string, err error) {
	if len(token) == 0 {
		return "", "", fmt.Errorf("provide Slack token")
	}

	if param == nil {
		return "", "", fmt.Errorf("provide filename and length")
	}

	if param.Filename == "" {
		return "", "", fmt.Errorf("provide filename")
	}

	if param.Length == 0 {
		return "", "", fmt.Errorf("provide length")
	}

	v := url.Values{}
	v.Set("filename", param.Filename)
	v.Set("length", strconv.Itoa(param.Length))

	if param.SnippetType != "" {
		v.Set("snippet_type", param.SnippetType)
	}

	req, err := http.NewRequest(http.MethodPost, filesGetUploadURLExternalURL, strings.NewReader(v.Encode()))
	if err != nil {
		return "", "", err
	}

	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read res.Body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to read res.Body and the status code: %d; body: %s", res.StatusCode, b)
	}

	apiRes := GetUploadURLExternalRes{}
	err = json.Unmarshal(b, &apiRes)
	if err != nil {
		return "", "", fmt.Errorf("response returned from slack is not json: body: %s: %w", b, err)
	}

	if !apiRes.OK {
		return "", "", fmt.Errorf("response has failed; body: %s", b)
	}

	return apiRes.UploadURL, apiRes.FileID, nil
}

func (c *Client) UploadToURL(ctx context.Context, fileName, uploadURL string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	contentType := writer.FormDataContentType()

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, uploadURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", contentType)

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read res.Body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to read res.Body and the status code: %d; body: %s", res.StatusCode, b)
	}

	return nil
}

type FileSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type CompleteUploadExternalRes struct {
	OK    bool `json:"ok"`
	Files []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"files"`
}

func (c *Client) CompleteUploadExternal(ctx context.Context, token, fileID, title string) error {
	if len(token) == 0 {
		return fmt.Errorf("provide Slack token")
	}

	request := []FileSummary{{ID: fileID, Title: title}}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return err
	}

	v := url.Values{}
	v.Set("files", string(requestBytes))

	req, err := http.NewRequest(http.MethodPost, filesCompleteUploadExternalURL, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read res.Body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to read res.Body and the status code: %d; body: %s", res.StatusCode, b)
	}

	apiRes := CompleteUploadExternalRes{}
	err = json.Unmarshal(b, &apiRes)
	if err != nil {
		return fmt.Errorf("response returned from slack is not json: body: %s: %w", b, err)
	}

	if !apiRes.OK {
		return fmt.Errorf("response has failed; body: %s", b)
	}

	return nil
}
