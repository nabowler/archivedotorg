package s3_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nabowler/archivedotorg/s3"
)

func MustClient(t *testing.T) s3.Client {
	if testing.Short() {
		t.Skipf("This test may take over 30 seconds")
	}

	api := s3.API{
		Key:    os.Getenv("S3_TEST_API_KEY"),
		Secret: os.Getenv("S3_TEST_API_SECRET"),
		URL:    os.Getenv("S3_TEST_API_URL"),
	}

	if api.Key == "" || api.Secret == "" {
		t.Skipf("API values not set. This integration test will be skipped")
	}

	return s3.Client{
		API: api,
		Client: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func TestUpload(t *testing.T) {
	client := MustClient(t)

	now := time.Now()
	opts := s3.UploadOptions{
		Upload:     strings.NewReader("This is a test"),
		FileName:   "test.txt",
		Identifier: fmt.Sprintf("s3-client-integration-test-%d", time.Now().UnixNano()),
		Title:      "this is the title",
		Description: `This is my description
		it has many

		lines.`,
		SubjectTags: []string{"subject1", "subject2"},
		Creator:     "TestUpload",
		Date:        &now,
		Metadata: s3.Metadata{
			"key1": []string{"value1", "value2"},
			"key2": []string{"value3"},
		},
		Collection:     s3.CollectionTest,
		AutoMakeBucket: true,
		KeepOldVersion: true,
		SkipDerive:     true,
	}
	err := client.Upload(context.Background(), opts)

	if err != nil {
		t.Fatalf("Unable to upload the integration test file: %v", err)
	}

	t.Logf("The test file can be found at https://archive.org/details/%s", opts.Identifier)
}

func TestFindIdentifier(t *testing.T) {
	if testing.Short() {
		t.Skipf("This test may take over several seconds")
	}
	c := s3.Client{}

	for _, ident := range []string{
		"foo",
		"The quick brown fox jumped over the lazy dogs",
		"This is a test 321   %wow!",
		"",
	} {
		ident := ident
		t.Run(fmt.Sprintf("FindIdentifier(%s)", ident), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			resp, err := c.FindIdentifier(ctx, ident)

			if err != nil {
				t.Fatalf("%v", err)
			}
			if !resp.Success {
				t.Error("Success was false")
			}

			t.Logf("%+v", resp)
		})
	}

}
