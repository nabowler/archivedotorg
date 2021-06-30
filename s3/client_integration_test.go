package s3_test

import (
	"context"
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
			Timeout: 30 * time.Second,
		},
	}
}

func TestUpload(t *testing.T) {
	client := MustClient(t)

	now := time.Now()
	err := client.Upload(context.Background(), s3.UploadOptions{
		Upload:      strings.NewReader("This is a test"),
		FileName:    "test.txt",
		Identifier:  "s3-client-integration-test-0981237645",
		Title:       "this is the title",
		SubjectTags: []string{"subject1", "subject2"},
		Creator:     "TestUpload",
		Date:        &now,
		Metadata: s3.Metadata{
			"key1": "value1",
			"key2": "value2",
		},
		Collection:     s3.CollectionData,
		AutoMakeBucket: true,
		KeepOldVersion: true,
	})

	if err != nil {
		t.Fatalf("Unable to upload the integration test file: %v", err)
	}
}
