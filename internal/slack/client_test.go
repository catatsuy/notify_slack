package slack_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	. "github.com/catatsuy/notify_slack/internal/slack"
)

func TestNewClient_badURL(t *testing.T) {
	_, err := NewClient("", nil)
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "client: missing url"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestNewClient_parsesURL(t *testing.T) {
	client, err := NewClient("https://example.com/foo/bar", nil)
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

	c, err := NewClient(testAPIServer.URL, nil)
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

	c, err := NewClient(testAPIServer.URL, nil)
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

	c, err := NewClientForPostFile(nil)
	if err != nil {
		t.Fatal(err)
	}

	err = c.PostFile(context.Background(), slackToken, param)

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

	c, err := NewClientForPostFile(nil)
	if err != nil {
		t.Fatal(err)
	}

	err = c.PostFile(context.Background(), slackToken, param)

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

	c, err := NewClientForPostFile(nil)
	if err != nil {
		t.Fatal(err)
	}

	err = c.PostFile(context.Background(), slackToken, param)

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

	c, err := NewClientForPostFile(nil)
	if err != nil {
		t.Fatal(err)
	}

	err = c.PostFile(context.Background(), slackToken, param)

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

	c, err := NewClientForPostFile(nil)
	if err != nil {
		t.Fatal(err)
	}

	err = c.PostFile(context.Background(), slackToken, param)

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := `response returned from slack is not json`
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}
