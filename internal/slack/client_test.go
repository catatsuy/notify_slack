package slack_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	. "github.com/catatsuy/notify_slack/internal/slack"
	"github.com/google/go-cmp/cmp"
)

func TestNewClient_badURL(t *testing.T) {
	_, err := NewClient("", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "client: missing url"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestNewClient_parsesURL(t *testing.T) {
	client, err := NewClient("https://example.com/foo/bar", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	expected := &url.URL{
		Scheme: "https",
		Host:   "example.com",
		Path:   "/foo/bar",
	}
	if !reflect.DeepEqual(client.URL, expected) {
		t.Fatalf("expected %q to equal %q", client.URL, expected)
	}
}

func TestPostText_Success(t *testing.T) {
	muxAPI := http.NewServeMux()
	testAPIServer := httptest.NewServer(muxAPI)
	defer testAPIServer.Close()

	param := &PostTextParam{
		Channel:   "test-channel",
		Username:  "tester",
		Text:      "testtesttest",
		IconEmoji: ":rocket:",
	}

	muxAPI.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		expectedType := "application/json"
		if contentType != expectedType {
			t.Fatalf("Content-Type expected %s, but %s", expectedType, contentType)
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()

		actualBody := &PostTextParam{}
		err = json.Unmarshal(bodyBytes, actualBody)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(actualBody, param) {
			t.Fatalf("expected %q to equal %q", actualBody, param)
		}

		http.ServeFile(w, r, "testdata/post_text_ok.html")
	})

	c, err := NewClient(testAPIServer.URL, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	err = c.PostText(context.Background(), param)

	if err != nil {
		t.Fatal(err)
	}
}

func TestPostText_Fail(t *testing.T) {
	muxAPI := http.NewServeMux()
	testAPIServer := httptest.NewServer(muxAPI)
	defer testAPIServer.Close()

	param := &PostTextParam{
		Channel:   "test2-channel",
		Username:  "tester",
		Text:      "testtesttest",
		IconEmoji: ":rocket:",
	}

	muxAPI.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		http.ServeFile(w, r, "testdata/post_text_fail.html")
	})

	c, err := NewClient(testAPIServer.URL, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	err = c.PostText(context.Background(), param)

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "status code: 404"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestPostFile_Success(t *testing.T) {
	muxAPI := http.NewServeMux()
	testAPIServer := httptest.NewServer(muxAPI)
	defer testAPIServer.Close()

	slackToken := "slack-token"

	param := &PostFileParam{
		Channel:  "test-channel",
		Content:  "testtesttest",
		Filename: "test.txt",
	}

	muxAPI.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		expectedType := "application/x-www-form-urlencoded"
		if contentType != expectedType {
			t.Fatalf("Content-Type expected %s, but %s", expectedType, contentType)
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()

		actualV, err := url.ParseQuery(string(bodyBytes))
		if err != nil {
			t.Fatal(err)
		}

		expectedV := url.Values{}
		expectedV.Set("token", slackToken)
		expectedV.Set("content", param.Content)
		expectedV.Set("filename", param.Filename)
		expectedV.Set("channels", param.Channel)

		if !reflect.DeepEqual(actualV, expectedV) {
			t.Fatalf("expected %q to equal %q", actualV, expectedV)
		}

		http.ServeFile(w, r, "testdata/post_files_upload_ok.json")
	})

	defer SetSlackFilesUploadURL(testAPIServer.URL)()

	c, err := NewClientForPostFile(slackToken, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	err = c.PostFile(context.Background(), param)

	if err != nil {
		t.Fatal(err)
	}
}

func TestPostFile_Success_provideFiletype(t *testing.T) {
	muxAPI := http.NewServeMux()
	testAPIServer := httptest.NewServer(muxAPI)
	defer testAPIServer.Close()

	slackToken := "slack-token"

	param := &PostFileParam{
		Channel:  "test-channel",
		Content:  "testtesttest",
		Filename: "test.txt",
		Filetype: "diff",
	}

	muxAPI.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		expectedType := "application/x-www-form-urlencoded"
		if contentType != expectedType {
			t.Fatalf("Content-Type expected %s, but %s", expectedType, contentType)
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()

		actualV, err := url.ParseQuery(string(bodyBytes))
		if err != nil {
			t.Fatal(err)
		}

		expectedV := url.Values{}
		expectedV.Set("token", slackToken)
		expectedV.Set("content", param.Content)
		expectedV.Set("filename", param.Filename)
		expectedV.Set("filetype", param.Filetype)
		expectedV.Set("channels", param.Channel)

		if !reflect.DeepEqual(actualV, expectedV) {
			t.Fatalf("expected %q to equal %q", actualV, expectedV)
		}

		http.ServeFile(w, r, "testdata/post_files_upload_ok.json")
	})

	defer SetSlackFilesUploadURL(testAPIServer.URL)()

	c, err := NewClientForPostFile(slackToken, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	err = c.PostFile(context.Background(), param)

	if err != nil {
		t.Fatal(err)
	}
}

func TestPostFile_FailNotOk(t *testing.T) {
	muxAPI := http.NewServeMux()
	testAPIServer := httptest.NewServer(muxAPI)
	defer testAPIServer.Close()

	slackToken := "slack-token"

	param := &PostFileParam{
		Channel:  "test-channel",
		Content:  "testtesttest",
		Filename: "test.txt",
	}

	muxAPI.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "testdata/post_files_upload_fail.json")
	})

	defer SetSlackFilesUploadURL(testAPIServer.URL)()

	c, err := NewClientForPostFile(slackToken, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	err = c.PostFile(context.Background(), param)

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `response has failed; body: {"ok":false,"error":"invalid_auth"}`
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestPostFile_FailNotResponseStatusCodeNotOK(t *testing.T) {
	muxAPI := http.NewServeMux()
	testAPIServer := httptest.NewServer(muxAPI)
	defer testAPIServer.Close()

	slackToken := "slack-token"

	param := &PostFileParam{
		Channel:  "test-channel",
		Content:  "testtesttest",
		Filename: "test.txt",
	}

	muxAPI.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		http.ServeFile(w, r, "testdata/post_files_upload_fail.json")
	})

	defer SetSlackFilesUploadURL(testAPIServer.URL)()

	c, err := NewClientForPostFile(slackToken, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	err = c.PostFile(context.Background(), param)

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `failed to read res.Body and the status code of the response from slack was not 200; body: {"ok":false,"error":"invalid_auth"}`
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestPostFile_FailNotJSON(t *testing.T) {
	muxAPI := http.NewServeMux()
	testAPIServer := httptest.NewServer(muxAPI)
	defer testAPIServer.Close()

	slackToken := "slack-token"

	param := &PostFileParam{
		Channel:  "test-channel",
		Content:  "testtesttest",
		Filename: "test.txt",
	}

	muxAPI.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "testdata/post_text_fail.html")
	})

	defer SetSlackFilesUploadURL(testAPIServer.URL)()

	c, err := NewClientForPostFile(slackToken, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	err = c.PostFile(context.Background(), param)

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `response returned from slack is not json`
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestUploadToURL_success(t *testing.T) {
	muxAPI := http.NewServeMux()
	testAPIServer := httptest.NewServer(muxAPI)
	defer testAPIServer.Close()

	muxAPI.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Expected multipart/form-data content type, got '%s'", r.Header.Get("Content-Type"))
		}

		err := r.ParseMultipartForm(32 << 10) // 32 KB
		if err != nil {
			t.Errorf("Error parsing multipart form: %v", err)
		}

		f, fh, err := r.FormFile("file")
		if err != nil {
			t.Errorf("Error retrieving file from form: %v", err)
		}

		if fh.Filename != "upload.txt" {
			t.Errorf("Expected filename 'testdata/upload_to_url_ok.txt', got '%s'", fh.Filename)
		}

		b, err := io.ReadAll(f)
		if err != nil {
			t.Errorf("Error reading file: %v", err)
		}

		expectedBody := []byte("this is test.\n")
		if !reflect.DeepEqual(b, expectedBody) {
			t.Errorf("expected %q to equal %q", b, expectedBody)
		}

		http.ServeFile(w, r, "testdata/upload_to_url_ok.txt")
	})

	c, err := NewClientForPostFile("abcd", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile("testdata/upload.txt")
	if err != nil {
		t.Fatal(err)
	}

	err = c.UploadToURL(context.Background(), "testdata/upload.txt", testAPIServer.URL, b)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUploadToURL_fail(t *testing.T) {
	muxAPI := http.NewServeMux()
	testAPIServer := httptest.NewServer(muxAPI)
	defer testAPIServer.Close()

	muxAPI.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	c, err := NewClientForPostFile("abcd", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile("testdata/upload.txt")
	if err != nil {
		t.Fatal(err)
	}

	err = c.UploadToURL(context.Background(), "upload.txt", testAPIServer.URL, b)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "status code: 400"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestCompleteUploadExternal_Success(t *testing.T) {
	muxAPI := http.NewServeMux()
	testAPIServer := httptest.NewServer(muxAPI)
	defer testAPIServer.Close()

	slackToken := "slack-token"

	muxAPI.HandleFunc("/api/files.completeUploadExternal", func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		expectedType := "application/x-www-form-urlencoded"
		if contentType != expectedType {
			t.Fatalf("Content-Type expected %s, but %s", expectedType, contentType)
		}

		authorization := r.Header.Get("Authorization")
		expectedAuth := "Bearer " + slackToken
		if authorization != expectedAuth {
			t.Fatalf("Authorization expected %s, but %s", expectedAuth, authorization)
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()

		actualV, err := url.ParseQuery(string(bodyBytes))
		if err != nil {
			t.Fatal(err)
		}

		expectedV := url.Values{}
		expectedV.Set("files", `[{"id":"file-id","title":"file-title"}]`)
		expectedV.Set("channel_id", "C0NF841BK")

		if diff := cmp.Diff(expectedV, actualV); diff != "" {
			t.Errorf("unexpected diff: (-want +got):\n%s", diff)
		}

		http.ServeFile(w, r, "testdata/files_complete_upload_external_ok.json")
	})

	defer SetFilesCompleteUploadExternalURL(testAPIServer.URL + "/api/files.completeUploadExternal")()

	c, err := NewClientForPostFile(slackToken, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	params := &CompleteUploadExternalParam{
		FileID:    "file-id",
		Title:     "file-title",
		ChannelID: "C0NF841BK",
	}
	err = c.CompleteUploadExternal(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
}
