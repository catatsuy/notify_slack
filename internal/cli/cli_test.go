package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/catatsuy/notify_slack/internal/config"
	"github.com/catatsuy/notify_slack/internal/slack"
	"github.com/google/go-cmp/cmp"
)

type fakeSlackClient struct {
	slack.Slack

	FakePostFile func(ctx context.Context, param *slack.PostFileParam, content []byte) error
}

func (c *fakeSlackClient) PostFile(ctx context.Context, param *slack.PostFileParam, content []byte) error {
	return c.FakePostFile(ctx, param, content)
}

func (c *fakeSlackClient) PostText(ctx context.Context, param *slack.PostTextParam) error {
	return nil
}

func TestRun_versionFlg(t *testing.T) {
	outStream, errStream, inputStream := new(bytes.Buffer), new(bytes.Buffer), new(bytes.Buffer)
	cl := NewCLI(outStream, errStream, inputStream, true)

	args := strings.Split("notify_slack -version", " ")
	status := cl.Run(args)

	if status != ExitCodeOK {
		t.Errorf("ExitStatus=%d, want %d", status, ExitCodeOK)
	}

	expected := fmt.Sprintf("notify_slack version %s", Version)
	if !strings.Contains(errStream.String(), expected) {
		t.Errorf("Output=%q, want %q", errStream.String(), expected)
	}
}

func TestUploadSnippet(t *testing.T) {
	cl := &CLI{
		sClient: &fakeSlackClient{},
		conf:    config.NewConfig(),
	}

	cl.conf.ChannelID = "C12345678"
	err := cl.uploadSnippet(t.Context(), "testdata/nofile.txt", "", "")
	want := "no such file or directory"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("error = %v; want %q", err, want)
	}

	cl.sClient = &fakeSlackClient{
		FakePostFile: func(ctx context.Context, param *slack.PostFileParam, content []byte) error {
			expectedFilename := "testdata/upload.txt"
			if param.Filename != expectedFilename {
				t.Errorf("expected %s; got %s", expectedFilename, param.Filename)
			}

			expectedContent := "upload_test\n"
			if diff := cmp.Diff(expectedContent, string(content)); diff != "" {
				t.Errorf("unexpected diff: (-want +got):\n%s", diff)
			}

			return nil
		},
	}

	err = cl.uploadSnippet(t.Context(), "testdata/upload.txt", "", "")
	if err != nil {
		t.Errorf("expected nil; got %v", err)
	}

	cl.sClient = &fakeSlackClient{
		FakePostFile: func(ctx context.Context, param *slack.PostFileParam, content []byte) error {
			expectedFilename := "overwrite.txt"
			if param.Filename != expectedFilename {
				t.Errorf("expected %s; got %s", expectedFilename, param.Filename)
			}

			expectedContent := "upload_test\n"
			if diff := cmp.Diff(expectedContent, string(content)); diff != "" {
				t.Errorf("unexpected diff: (-want +got):\n%s", diff)
			}

			return nil
		},
	}

	err = cl.uploadSnippet(t.Context(), "testdata/upload.txt", "overwrite.txt", "")
	if err != nil {
		t.Errorf("expected nil; got %v", err)
	}

	cl.sClient = &fakeSlackClient{
		FakePostFile: func(ctx context.Context, param *slack.PostFileParam, content []byte) error {
			if param.ChannelID != cl.conf.ChannelID {
				t.Errorf("expected %s; got %s", cl.conf.ChannelID, param.ChannelID)
			}

			expectedFilename := "overwrite.txt"
			if param.Filename != expectedFilename {
				t.Errorf("expected %s; got %s", expectedFilename, param.Filename)
			}

			expectedSnippetType := "diff"
			if param.SnippetType != expectedSnippetType {
				t.Errorf("expected %s; got %s", expectedSnippetType, param.SnippetType)
			}

			expectedContent := "upload_test\n"
			if diff := cmp.Diff(expectedContent, string(content)); diff != "" {
				t.Errorf("unexpected diff: (-want +got):\n%s", diff)
			}

			return nil
		},
	}

	err = cl.uploadSnippet(t.Context(), "testdata/upload.txt", "overwrite.txt", "diff")
	if err != nil {
		t.Errorf("expected nil; got %v", err)
	}

	t.Run("rejects oversized file", func(t *testing.T) {
		oldLimit := maxSnippetBytes
		maxSnippetBytes = 10
		t.Cleanup(func() {
			maxSnippetBytes = oldLimit
		})

		largeFile, err := os.CreateTemp(t.TempDir(), "oversize-*.txt")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		oversized := make([]byte, int(maxSnippetBytes)+1)
		if _, err := largeFile.Write(oversized); err != nil {
			t.Fatalf("failed to write oversize data: %v", err)
		}
		if err := largeFile.Close(); err != nil {
			t.Fatalf("failed to close oversize temp file: %v", err)
		}

		if err := cl.uploadSnippet(t.Context(), largeFile.Name(), "", ""); err == nil {
			t.Fatal("expected error for oversize upload, got nil")
		} else if !strings.Contains(err.Error(), "capped") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
