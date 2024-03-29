package web_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/nabowler/archivedotorg/web"
)

func TestSave(t *testing.T) {
	client := web.Client{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	link, err := url.Parse("https://reddit.com")
	if err != nil {
		t.Fatalf("Cannot parse url: %v", err)
	}

	result, err := client.Save(context.Background(), link, web.SaveOptions{
		SaveOutLinks: true,
	})
	if result != nil {
		defer result.Body.Close()
	}

	if err != nil {
		t.Fatalf("Unable to save link: %v", err)
	}

	t.Logf("Result: %+v", result)
}
