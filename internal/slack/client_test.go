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
		expectedV.Set("length", strconv.FormatInt(param.Length, 10))

		if diff := cmp.Diff(expectedV, actualV); diff != "" {
			t.Errorf("unexpected diff: (-want +got):\n%s", diff)
		}

		http.ServeFile(w, r, "testdata/files_get_upload_url_external_ok.json")
	})

	defer SetFilesGetUploadURLExternalURL(testAPIServer.URL + "/api/files.getUploadURLExternal")()

	c, err := NewClientForFile(slackToken)
	if err != nil {
		t.Fatal(err)
	}

	uploadURL, fileID, err := c.GetUploadURLExternalURL(context.Background(), param)
	if err != nil {
		t.Fatal(err)
	}

	expectedUploadURL := "https://files.slack.com/upload/v1/ABC123456"
	if uploadURL != expectedUploadURL {
		t.Fatalf("expected %q to equal %q", uploadURL, expectedUploadURL)
	}

	expectedFileID := "F123ABC456"
	if fileID != expectedFileID {
		t.Fatalf("expected %q to equal %q", fileID, expectedFileID)
	}
}

func TestPostFile_FailCallFunc(t *testing.T) {
	muxAPI := http.NewServeMux()
	testAPIServer := httptest.NewServer(muxAPI)
	defer testAPIServer.Close()

	slackToken := "slack-token"

	muxAPI.HandleFunc("/api/files.getUploadURLExternal", func(w http.ResponseWriter, r *http.Request) {
		panic("unexpected call")
	})

	defer SetFilesGetUploadURLExternalURL(testAPIServer.URL + "/api/files.getUploadURLExternal")()

	_, err := NewClientForFile("")
	expectedErrorPart := "provide Slack token"
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	} else if !strings.Contains(err.Error(), expectedErrorPart) {
		t.Fatalf("expected %q to contain %q", err.Error(), expectedErrorPart)
	}

	c, err := NewClientForFile(slackToken)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = c.GetUploadURLExternalURL(context.Background(), nil)
	expectedErrorPart = "provide filename and length"
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	} else if !strings.Contains(err.Error(), expectedErrorPart) {
		t.Fatalf("expected %q to contain %q", err.Error(), expectedErrorPart)
	}

	_, _, err = c.GetUploadURLExternalURL(context.Background(), &GetUploadURLExternalResParam{})
	expectedErrorPart = "provide filename"
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	} else if !strings.Contains(err.Error(), expectedErrorPart) {
		t.Fatalf("expected %q to contain %q", err.Error(), expectedErrorPart)
	}

	_, _, err = c.GetUploadURLExternalURL(context.Background(), &GetUploadURLExternalResParam{Filename: "test.txt"})
	expectedErrorPart = "provide length"
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	} else if !strings.Contains(err.Error(), expectedErrorPart) {
		t.Fatalf("expected %q to contain %q", err.Error(), expectedErrorPart)
	}
}

func TestPostFile_FailAPINotOK(t *testing.T) {
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
		expectedV.Set("length", strconv.FormatInt(param.Length, 10))

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

	c, err := NewClientForFile(slackToken)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = c.GetUploadURLExternalURL(context.Background(), param)

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

func TestPostFile_FailAPIStatusOK(t *testing.T) {
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
		expectedV.Set("length", strconv.FormatInt(param.Length, 10))

		if diff := cmp.Diff(expectedV, actualV); diff != "" {
			t.Errorf("unexpected diff: (-want +got):\n%s", diff)
		}

		w.Header().Set("Content-Type", "application/json")

		b, err := os.ReadFile("testdata/files_get_upload_url_external_fail_invalid_arguments.json")
		if err != nil {
			t.Fatal(err)
		}

		w.Write(b)
	})

	defer SetFilesGetUploadURLExternalURL(testAPIServer.URL + "/api/files.getUploadURLExternal")()

	c, err := NewClientForFile(slackToken)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = c.GetUploadURLExternalURL(context.Background(), param)

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	} else {
		expected := "response has failed"
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("expected %q to contain %q", err.Error(), expected)
		}

		expectedBodyPart := `"invalid_arguments"`
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
		expectedV.Set("length", strconv.FormatInt(param.Length, 10))

		if diff := cmp.Diff(expectedV, actualV); diff != "" {
			t.Errorf("unexpected diff: (-want +got):\n%s", diff)
		}

		w.Header().Set("Content-Type", "text/plain")

		w.Write([]byte("this is not json"))
	})

	defer SetFilesGetUploadURLExternalURL(testAPIServer.URL + "/api/files.getUploadURLExternal")()

	c, err := NewClientForFile(slackToken)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = c.GetUploadURLExternalURL(context.Background(), param)

	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	} else {
		expectedBodyPart := `this is not json`
		if !strings.Contains(err.Error(), expectedBodyPart) {
			t.Errorf("expected %q to contain %q", err.Error(), expectedBodyPart)
		}
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

	c, err := NewClientForFile("abcd")
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open("testdata/upload.txt")
	if err != nil {
		t.Fatal(err)
	}

	err = c.UploadToURL(context.Background(), "testdata/upload.txt", testAPIServer.URL, f)
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

	c, err := NewClientForFile("abcd")
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open("testdata/upload.txt")
	if err != nil {
		t.Fatal(err)
	}

	err = c.UploadToURL(context.Background(), "upload.txt", testAPIServer.URL, f)
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

		if diff := cmp.Diff(expectedV, actualV); diff != "" {
			t.Errorf("unexpected diff: (-want +got):\n%s", diff)
		}

		http.ServeFile(w, r, "testdata/files_complete_upload_external_ok.json")
	})

	defer SetFilesCompleteUploadExternalURL(testAPIServer.URL + "/api/files.completeUploadExternal")()

	c, err := NewClientForFile(slackToken)
	if err != nil {
		t.Fatal(err)
	}

	err = c.CompleteUploadExternal(context.Background(), "file-id", "file-title")
	if err != nil {
		t.Fatal(err)
	}
}
