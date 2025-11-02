package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
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

	Token string

	Logger *slog.Logger
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

type PostFileParam struct {
	ChannelID   string
	Filename    string
	AltText     string
	Title       string
	SnippetType string
}

type GetUploadURLExternalResParam struct {
	Filename    string
	Length      int
	SnippetType string
	AltText     string
}

type Slack interface {
	PostText(ctx context.Context, param *PostTextParam) error
	PostFile(ctx context.Context, param *PostFileParam, content []byte) error
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

	req, err := c.newRequest(ctx, http.MethodPost, bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	c.Logger.Debug("request",
		slog.String("url", req.URL.String()),
		slog.String("method", req.Method),
		slog.Any("header", sanitizeHeaders(req.Header)),
	)

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read res.Body: %w", err)
	}

	c.Logger.Debug("request",
		slog.String("url", req.URL.String()),
		slog.String("method", req.Method),
		slog.Any("header", sanitizeHeaders(req.Header)),
		slog.Int("status", res.StatusCode),
		slog.String("body", string(body)),
	)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d; body: %s", res.StatusCode, body)
	}

	return nil
}

func (c *Client) PostFile(ctx context.Context, param *PostFileParam, content []byte) error {
	uParam := &GetUploadURLExternalResParam{
		Filename:    param.Filename,
		Length:      len(content),
		SnippetType: param.SnippetType,
		AltText:     param.AltText,
	}

	uploadURL, fileID, err := c.GetUploadURLExternalURL(ctx, uParam)
	if err != nil {
		return fmt.Errorf("failed to get upload url: %w", err)
	}

	err = c.UploadToURL(ctx, param.Filename, uploadURL, content)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	cParam := &CompleteUploadExternalParam{
		FileID:    fileID,
		Title:     param.Title,
		ChannelID: param.ChannelID,
	}

	err = c.CompleteUploadExternal(ctx, cParam)
	if err != nil {
		return fmt.Errorf("failed to complete upload: %w", err)
	}

	return nil
}

func (c *Client) GetUploadURLExternalURL(ctx context.Context, param *GetUploadURLExternalResParam) (uploadURL string, fileID string, err error) {
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

	if param.AltText != "" {
		v.Set("alt_text", param.AltText)
	}

	if param.SnippetType != "" {
		v.Set("snippet_type", param.SnippetType)
	}

	req, err := http.NewRequest(http.MethodPost, filesGetUploadURLExternalURL, strings.NewReader(v.Encode()))
	if err != nil {
		return "", "", err
	}

	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

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

func (c *Client) UploadToURL(ctx context.Context, filename, uploadURL string, content []byte) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = part.Write(content)
	if err != nil {
		return fmt.Errorf("failed to write content: %w", err)
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
	Title string `json:"title,omitempty"`
}

type CompleteUploadExternalRes struct {
	OK    bool `json:"ok"`
	Files []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"files"`
}

type CompleteUploadExternalParam struct {
	FileID    string
	Title     string
	ChannelID string
}

func (c *Client) CompleteUploadExternal(ctx context.Context, params *CompleteUploadExternalParam) error {
	request := []FileSummary{{ID: params.FileID, Title: params.Title}}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return err
	}

	v := url.Values{}
	v.Set("files", string(requestBytes))
	if params.ChannelID != "" {
		v.Set("channel_id", params.ChannelID)
	}

	req, err := http.NewRequest(http.MethodPost, filesCompleteUploadExternalURL, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read res.Body: %w", err)
	}

	c.Logger.Debug("request",
		slog.String("url", req.URL.String()),
		slog.String("method", req.Method),
		slog.Any("header", sanitizeHeaders(req.Header)),
		slog.Int("status", res.StatusCode),
		slog.String("body", string(b)),
	)

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

func sanitizeHeaders(header http.Header) http.Header {
	if header == nil {
		return nil
	}

	sanitized := header.Clone()

	for key := range sanitized {
		if isSensitiveHeader(key) {
			values := sanitized[key]
			for i := range values {
				values[i] = maskSensitiveValue(values[i])
			}
		}
	}

	return sanitized
}

func isSensitiveHeader(headerKey string) bool {
	return strings.EqualFold(headerKey, "Authorization")
}

func maskSensitiveValue(value string) string {
	if value == "" {
		return value
	}

	const placeholder = "[redacted]"

	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return "Bearer " + placeholder
	}

	return placeholder
}
