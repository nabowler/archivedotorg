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
	if testing.Short() {
		t.Skipf("This test may take over 30 seconds")
	}
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
			"key1": "value1",
			"key2": "value2",
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
