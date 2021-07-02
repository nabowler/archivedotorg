package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type (
	Client struct {
		Client *http.Client
		API    API
	}

	API struct {
		Key    string
		Secret string
		URL    string
	}

	UploadOptions struct {
		Upload      io.Reader
		FileName    string
		Identifier  string
		Title       string
		Description string
		SubjectTags []string
		Creator     string
		Date        *time.Time
		Metadata    Metadata
		Collection  Collection

		AutoMakeBucket bool
		KeepOldVersion bool
		SkipDerive     bool
		// TODO
		// Language string
		// License string
		// MediaType string
	}

	Metadata map[string]string

	Collection string

	IdentifierResponse struct {
		Identifier string `json:"identifier"`
		Success    bool   `json:"success"`
	}
)

const (
	DefaultURL      = "https://s3.us.archive.org"
	uploadURLFormat = "%s/%s/%s"

	CollectionData   Collection = "opensource_media"
	CollectionMovies Collection = "opensource_movies"
	CollectionTest   Collection = "test_collection"
	// TODO: other collections
)

func (c Client) Upload(ctx context.Context, opts UploadOptions) error {
	// TODO: read https://archive.org/services/docs/api/ias3.html

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.uploadURL(opts), opts.Upload)
	if err != nil {
		return fmt.Errorf("Unable to create the request: %w", err)
	}

	for k, v := range opts.Metadata {
		if !strings.HasPrefix(k, "x-amz-meta") {
			k = "x-amz-meta-" + k
		}
		req.Header.Set(k, uriEncode(v))
	}
	for i, v := range opts.SubjectTags {
		req.Header.Set(fmt.Sprintf("x-amz-meta%02d-subject", i), uriEncode(v))
	}

	req.Header.Set("authorization", fmt.Sprintf("LOW %s:%s", c.API.Key, c.API.Secret))
	if opts.Collection == "" {
		opts.Collection = CollectionData
	}
	req.Header.Set("x-archive-meta01-collection", string(opts.Collection))
	if opts.Title != "" {
		req.Header.Set("x-archive-meta-title", uriEncode(opts.Title))
	}
	if opts.Date != nil {
		req.Header.Set("x-archive-meta01-date", uriEncode(opts.Date.Format("2006-01-02")))
	}
	if opts.Description != "" {
		req.Header.Set("x-archive-meta01-description", uriEncode(opts.Description))
	}
	if opts.Creator != "" {
		req.Header.Set("x-archive-meta01-creator", uriEncode(opts.Creator))
	}

	req.Header.Set("x-archive-meta01-scanner", uriEncode("archivedotorg/s3"))

	if opts.AutoMakeBucket {
		req.Header.Set("x-amz-auto-make-bucket", "1")
	}
	if opts.KeepOldVersion {
		req.Header.Set("x-archive-keep-old-version", "1")
	}
	if opts.SkipDerive {
		req.Header.Set("x-archive-queue-derive", "0")
	}

	// for certain types of uploads, we can provide a size hint
	var sizeHint int64
	switch t := opts.Upload.(type) {
	case *bytes.Buffer:
		sizeHint = int64(t.Len())
	case *bytes.Reader:
		sizeHint = int64(t.Len())
	case *strings.Reader:
		sizeHint = int64(t.Len())
	case *os.File:
		if stat, err := t.Stat(); stat != nil && err == nil {
			sizeHint = stat.Size()
		}
	}
	if sizeHint > 0 {
		req.Header.Set("x-archive-size-hint", fmt.Sprintf("%d", sizeHint))
	}

	req.Header.Set("x-amz-acl", "bucket-owner-full-control")

	//TODO: --header 'x-archive-meta-mediatype:images'
	//      --header 'x-archive-meta-language:eng'
	//      --header 'x-archive-meta01-licenseurl:uri(http%3A%2F%2Fcreativecommons.org%2Fpublicdomain%2Fmark%2F1.0%2F)'

	httpClient := c.Client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Unable to read the %d response body: %w", resp.StatusCode, err)
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%d Response: %s", resp.StatusCode, string(body))
	}

	// TODO: Handle response
	//  - Slow down
	//  - Anything of value in the body?
	//    - On error, yes. On success, no.

	return nil
}

func (c Client) FindIdentifier(ctx context.Context, identifier string) (IdentifierResponse, error) {
	var ret IdentifierResponse
	req, err := http.NewRequest("POST", "https://archive.org/upload/app/upload_api.php", strings.NewReader(url.Values{
		"name":       []string{"identifierAvailable"},
		"identifier": []string{identifier},
		"findUnique": []string{"true"},
	}.Encode()))
	if err != nil {
		return ret, fmt.Errorf("Unable to build the request: %w", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	httpClient := c.Client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return ret, fmt.Errorf("Unable to perform the request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ret, fmt.Errorf("Unable to read the response: %w", err)
	}

	return ret, json.Unmarshal(body, &ret)
}

func (c Client) CheckLimits(ctx context.Context, opts UploadOptions) (interface{}, error) {
	// https://archive.org/services/docs/api/ias3.html#use-limits
	return nil, fmt.Errorf("Not implemented")
}

func (c Client) uploadURL(opts UploadOptions) string {
	baseURL := c.API.URL
	if baseURL == "" {
		baseURL = DefaultURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return fmt.Sprintf(uploadURLFormat, baseURL, opts.identifier(), opts.FileName)
}

func (opts UploadOptions) identifier() string {
	if opts.Identifier != "" {
		return opts.Identifier
	}
	// i'm guessing at these rules. seems like lowercase alphanumeric and hyphens is the rule
	// suggested regex: ^[a-zA-Z0-9][a-zA-Z0-9_.-]{4,100}$

	ident := strings.ToLower(opts.Title)
	ident = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r
		}
		if r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, ident)
	return ident
}

func uriEncode(s string) string {
	return fmt.Sprintf("uri(%s)", url.QueryEscape(s))
}
