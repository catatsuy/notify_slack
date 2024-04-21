package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/catatsuy/notify_slack/internal/config"
	"github.com/catatsuy/notify_slack/internal/slack"
	"github.com/google/go-cmp/cmp"
)

type fakeSlackClient struct {
	slack.Slack

	FakePostFile func(ctx context.Context, params *slack.PostFileParam, content []byte) error
}

func (c *fakeSlackClient) PostFile(ctx context.Context, params *slack.PostFileParam, content []byte) error {
	return c.FakePostFile(ctx, params, content)
}

func (c *fakeSlackClient) PostText(ctx context.Context, param *slack.PostTextParam) error {
	return nil
}

func TestRun_versionFlg(t *testing.T) {
	outStream, errStream, inputStream := new(bytes.Buffer), new(bytes.Buffer), new(bytes.Buffer)
	cl := NewCLI(outStream, errStream, inputStream)

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

	cl.conf.FileChannelID = "C12345678"
	err := cl.uploadSnippet(context.Background(), "testdata/nofile.txt", "", "")
	want := "no such file or directory"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("error = %v; want %q", err, want)
	}

	cl.sClient = &fakeSlackClient{
		FakePostFile: func(ctx context.Context, params *slack.PostFileParam, content []byte) error {
			expectedFilename := "testdata/upload.txt"
			if params.Filename != expectedFilename {
				t.Errorf("expected %s; got %s", expectedFilename, params.Filename)
			}

			expectedContent := "upload_test\n"
			if diff := cmp.Diff(expectedContent, string(content)); diff != "" {
				t.Errorf("unexpected diff: (-want +got):\n%s", diff)
			}

			return nil
		},
	}

	err = cl.uploadSnippet(context.Background(), "testdata/upload.txt", "", "")
	if err != nil {
		t.Errorf("expected nil; got %v", err)
	}

	cl.sClient = &fakeSlackClient{
		FakePostFile: func(ctx context.Context, params *slack.PostFileParam, content []byte) error {
			expectedFilename := "overwrite.txt"
			if params.Filename != expectedFilename {
				t.Errorf("expected %s; got %s", expectedFilename, params.Filename)
			}

			expectedContent := "upload_test\n"
			if diff := cmp.Diff(expectedContent, string(content)); diff != "" {
				t.Errorf("unexpected diff: (-want +got):\n%s", diff)
			}

			return nil
		},
	}

	err = cl.uploadSnippet(context.Background(), "testdata/upload.txt", "overwrite.txt", "")
	if err != nil {
		t.Errorf("expected nil; got %v", err)
	}

	cl.sClient = &fakeSlackClient{
		FakePostFile: func(ctx context.Context, params *slack.PostFileParam, content []byte) error {
			if params.ChannelID != cl.conf.FileChannelID {
				t.Errorf("expected %s; got %s", cl.conf.FileChannelID, params.ChannelID)
			}

			expectedFilename := "overwrite.txt"
			if params.Filename != expectedFilename {
				t.Errorf("expected %s; got %s", expectedFilename, params.Filename)
			}

			expectedSnippetType := "diff"
			if params.SnippetType != expectedSnippetType {
				t.Errorf("expected %s; got %s", expectedSnippetType, params.SnippetType)
			}

			expectedContent := "upload_test\n"
			if diff := cmp.Diff(expectedContent, string(content)); diff != "" {
				t.Errorf("unexpected diff: (-want +got):\n%s", diff)
			}

			return nil
		},
	}

	err = cl.uploadSnippet(context.Background(), "testdata/upload.txt", "overwrite.txt", "diff")
	if err != nil {
		t.Errorf("expected nil; got %v", err)
	}
}
