package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/catatsuy/notify_slack/config"
	"github.com/catatsuy/notify_slack/slack"
)

type fakeSlackClient struct {
	slack.Slack

	FakePostFile func(ctx context.Context, token string, param *slack.PostFileParam) error
}

func (c *fakeSlackClient) PostFile(ctx context.Context, token string, param *slack.PostFileParam) error {
	return c.FakePostFile(ctx, token, param)
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

func TestRun_providedStdin(t *testing.T) {
	errStream, inputStream := new(bytes.Buffer), new(bytes.Buffer)

	// cf: https://cs.opensource.google/go/x/term/+/master:term_test.go
	if runtime.GOOS != "linux" {
		t.Skipf("unknown terminal path for GOOS %v", runtime.GOOS)
	}
	file, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var outStream io.Writer = file

	cl := NewCLI(outStream, errStream, inputStream)

	args := strings.Split("notify_slack", " ")
	status := cl.Run(args)

	if status != ExitCodeFail {
		t.Errorf("ExitStatus=%d, want %d", status, ExitCodeOK)
	}

	expected := "No input file specified"
	if !strings.Contains(errStream.String(), expected) {
		t.Errorf("Output=%q, want %q", errStream.String(), expected)
	}
}

func TestUploadSnippet(t *testing.T) {
	cl := &CLI{
		sClient: &fakeSlackClient{},
		conf:    config.NewConfig(),
	}

	err := cl.uploadSnippet(context.Background(), "", "", "")
	want := "must specify channel"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("error = %v; want %q", err, want)
	}

	cl.conf.Channel = "normal_channel"
	err = cl.uploadSnippet(context.Background(), "testdata/nofile.txt", "", "")
	want = "no such file or directory"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("error = %v; want %q", err, want)
	}

	cl.sClient = &fakeSlackClient{
		FakePostFile: func(ctx context.Context, token string, param *slack.PostFileParam) error {
			if param.Channel != cl.conf.Channel {
				t.Errorf("expected %s; got %s", cl.conf.Channel, param.Channel)
			}

			expectedFilename := "testdata/upload.txt"
			if param.Filename != expectedFilename {
				t.Errorf("expected %s; got %s", expectedFilename, param.Filename)
			}

			expectedContent := "upload_test\n"
			if param.Content != expectedContent {
				t.Errorf("expected %q; got %q", expectedContent, param.Content)
			}

			return nil
		},
	}

	err = cl.uploadSnippet(context.Background(), "testdata/upload.txt", "", "")
	if err != nil {
		t.Errorf("expected nil; got %v", err)
	}

	cl.sClient = &fakeSlackClient{
		FakePostFile: func(ctx context.Context, token string, param *slack.PostFileParam) error {
			if param.Channel != cl.conf.Channel {
				t.Errorf("expected %s; got %s", cl.conf.Channel, param.Channel)
			}

			expectedFilename := "overwrite.txt"
			if param.Filename != expectedFilename {
				t.Errorf("expected %s; got %s", expectedFilename, param.Filename)
			}

			expectedContent := "upload_test\n"
			if param.Content != expectedContent {
				t.Errorf("expected %q; got %q", expectedContent, param.Content)
			}

			return nil
		},
	}

	err = cl.uploadSnippet(context.Background(), "testdata/upload.txt", "overwrite.txt", "")
	if err != nil {
		t.Errorf("expected nil; got %v", err)
	}

	cl.sClient = &fakeSlackClient{
		FakePostFile: func(ctx context.Context, token string, param *slack.PostFileParam) error {
			if param.Channel != cl.conf.Channel {
				t.Errorf("expected %s; got %s", cl.conf.Channel, param.Channel)
			}

			expectedFilename := "overwrite.txt"
			if param.Filename != expectedFilename {
				t.Errorf("expected %s; got %s", expectedFilename, param.Filename)
			}

			expectedContent := "upload_test\n"
			if param.Content != expectedContent {
				t.Errorf("expected %q; got %q", expectedContent, param.Content)
			}

			expectedFiletype := "diff"
			if param.Filetype != expectedFiletype {
				t.Errorf("expected %s; got %s", expectedFiletype, param.Filetype)
			}

			return nil
		},
	}

	err = cl.uploadSnippet(context.Background(), "testdata/upload.txt", "overwrite.txt", "diff")
	if err != nil {
		t.Errorf("expected nil; got %v", err)
	}

	cl.conf.SnippetChannel = "snippet_channel"

	cl.sClient = &fakeSlackClient{
		FakePostFile: func(ctx context.Context, token string, param *slack.PostFileParam) error {
			if param.Channel != cl.conf.SnippetChannel {
				t.Errorf("expected %s; got %s", cl.conf.SnippetChannel, param.Channel)
			}

			expectedFilename := "testdata/upload.txt"
			if param.Filename != expectedFilename {
				t.Errorf("expected %s; got %s", expectedFilename, param.Filename)
			}

			expectedContent := "upload_test\n"
			if param.Content != expectedContent {
				t.Errorf("expected %q; got %q", expectedContent, param.Content)
			}

			return nil
		},
	}

	err = cl.uploadSnippet(context.Background(), "testdata/upload.txt", "", "")
	if err != nil {
		t.Errorf("expected nil; got %v", err)
	}

	cl.conf.PrimaryChannel = "primary_channel"

	cl.sClient = &fakeSlackClient{
		FakePostFile: func(ctx context.Context, token string, param *slack.PostFileParam) error {
			if param.Channel != cl.conf.PrimaryChannel {
				t.Errorf("expected %s; got %s", cl.conf.PrimaryChannel, param.Channel)
			}

			expectedFilename := "testdata/upload.txt"
			if param.Filename != expectedFilename {
				t.Errorf("expected %s; got %s", expectedFilename, param.Filename)
			}

			expectedContent := "upload_test\n"
			if param.Content != expectedContent {
				t.Errorf("expected %q; got %q", expectedContent, param.Content)
			}

			return nil
		},
	}

	err = cl.uploadSnippet(context.Background(), "testdata/upload.txt", "", "")
	if err != nil {
		t.Errorf("expected nil; got %v", err)
	}
}
