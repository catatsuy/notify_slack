package slack_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	. "github.com/catatsuy/notify_slack/internal/slack"
	"github.com/google/go-cmp/cmp"
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
		b, err := os.ReadFile("testdata/post_text_fail.html")
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusNotFound)
		w.Write(b)
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

	param := &GetUploadURLExternalResParam{
		Filename: "test.txt",
		Length:   100,
	}

	muxAPI.HandleFunc("/api/files.getUploadURLExternal", func(w http.ResponseWriter, r *http.Request) {
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
		expectedV.Set("filename", param.Filename)
		expectedV.Set("length", strconv.Itoa(param.Length))

		if diff := cmp.Diff(expectedV, actualV); diff != "" {
			t.Errorf("unexpected diff: (-want +got):\n%s", diff)
		}

		http.ServeFile(w, r, "testdata/files_get_upload_url_external_ok.json")
	})

	defer SetFilesGetUploadURLExternalURL(testAPIServer.URL + "/api/files.getUploadURLExternal")()

	c, err := NewClientForFile(nil)
	if err != nil {
		t.Fatal(err)
	}

	err = c.GetUploadURLExternalURL(context.Background(), slackToken, param)

	if err != nil {
		t.Fatal(err)
	}
}

func TestPostFile_FailAPI(t *testing.T) {
	muxAPI := http.NewServeMux()
	testAPIServer := httptest.NewServer(muxAPI)
	defer testAPIServer.Close()

	slackToken := "slack-token"

	param := &GetUploadURLExternalResParam{
		Filename: "test.txt",
		Length:   100,
	}

	muxAPI.HandleFunc("/api/files.getUploadURLExternal", func(w http.ResponseWriter, r *http.Request) {
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
		expectedV.Set("filename", param.Filename)
		expectedV.Set("length", strconv.Itoa(param.Length))

		if diff := cmp.Diff(expectedV, actualV); diff != "" {
			t.Errorf("unexpected diff: (-want +got):\n%s", diff)
		}

		w.WriteHeader(http.StatusForbidden)
		w.Header().Set("Content-Type", "application/json")

		b, err := os.ReadFile("testdata/files_get_upload_url_external_fail.json")
		if err != nil {
			t.Fatal(err)
		}

		w.Write(b)
	})

	defer SetFilesGetUploadURLExternalURL(testAPIServer.URL + "/api/files.getUploadURLExternal")()

	c, err := NewClientForFile(nil)
	if err != nil {
		t.Fatal(err)
	}

	err = c.GetUploadURLExternalURL(context.Background(), "", param)
	expectedErrorPart := "provide Slack token"
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	} else if !strings.Contains(err.Error(), expectedErrorPart) {
		t.Fatalf("expected %q to contain %q", err.Error(), expectedErrorPart)
	}

	err = c.GetUploadURLExternalURL(context.Background(), slackToken, param)

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	} else {
		expected := "status code: 403"
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("expected %q to contain %q", err.Error(), expected)
		}

		expectedBodyPart := `"invalid_auth"`
		if !strings.Contains(err.Error(), expectedBodyPart) {
			t.Errorf("expected %q to contain %q", err.Error(), expectedBodyPart)
		}
	}
}

func TestPostFile_FailBrokenJSON(t *testing.T) {
	muxAPI := http.NewServeMux()
	testAPIServer := httptest.NewServer(muxAPI)
	defer testAPIServer.Close()

	slackToken := "slack-token"

	param := &GetUploadURLExternalResParam{
		Filename: "test.txt",
		Length:   100,
	}

	muxAPI.HandleFunc("/api/files.getUploadURLExternal", func(w http.ResponseWriter, r *http.Request) {
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
		expectedV.Set("filename", param.Filename)
		expectedV.Set("length", strconv.Itoa(param.Length))

		if diff := cmp.Diff(expectedV, actualV); diff != "" {
			t.Errorf("unexpected diff: (-want +got):\n%s", diff)
		}

		w.Header().Set("Content-Type", "text/plain")

		w.Write([]byte("this is not json"))
	})

	defer SetFilesGetUploadURLExternalURL(testAPIServer.URL + "/api/files.getUploadURLExternal")()

	c, err := NewClientForFile(nil)
	if err != nil {
		t.Fatal(err)
	}

	err = c.GetUploadURLExternalURL(context.Background(), slackToken, param)

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	} else {
		expectedBodyPart := `this is not json`
		if !strings.Contains(err.Error(), expectedBodyPart) {
			t.Errorf("expected %q to contain %q", err.Error(), expectedBodyPart)
		}
	}
}