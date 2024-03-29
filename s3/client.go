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
		Scanner     string

		AutoMakeBucket  bool
		KeepOldVersion  bool
		SkipDerive      bool
		SkipUniqueCheck bool
		// TODO
		// Language string
		// License string
		// MediaType string
	}

	Metadata map[string][]string

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

	// determine the identifier
	ident := opts.identifier()
	if !opts.SkipUniqueCheck {
		identifier, err := c.FindIdentifier(ctx, opts.identifier())
		if err != nil {
			return fmt.Errorf("Unable to find a unique identifier: %w", err)
		}
		if !identifier.Success {
			return fmt.Errorf("Finding a unique identifier failed without an error.")
		}
		ident = identifier.Identifier
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.uploadURL(opts, ident), opts.Upload)
	if err != nil {
		return fmt.Errorf("Unable to create the request: %w", err)
	}

	for k, v := range opts.Metadata {
		for i, vi := range v {
			req.Header.Set(fmt.Sprintf("x-amz-meta%02d-%s", i, k), uriEncode(vi))
		}
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

	scanner := opts.Scanner
	if scanner == "" {
		scanner = "archivedotorg/s3"
	}
	req.Header.Set("x-archive-meta01-scanner", uriEncode(scanner))

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
	/*
		503 Response:
		    <?xml version='1.0' encoding='UTF-8'?>
			<Error>
			  <Code>SlowDown</Code>
			  <Message>Please reduce your request rate.</Message>
			  <Resource>Your upload of s3-client-integration-multi-test-1635689404660360760 from username red@ac.ted appears to be spam. If you believe this is a mistake, contact info@archive.org and include this entire message in your email.</Resource>
			  <RequestId>cd36a624-c520-4d46-8a0b-1c476ee49f2e</RequestId>
			</Error>
	*/

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

func (c Client) uploadURL(opts UploadOptions, identifier string) string {
	baseURL := c.API.URL
	if baseURL == "" {
		baseURL = DefaultURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return fmt.Sprintf(uploadURLFormat, baseURL, identifier, opts.FileName)
}

func (opts UploadOptions) identifier() string {
	// suggested regex: ^[a-zA-Z0-9][a-zA-Z0-9_.-]{4,100}$

	if opts.Identifier != "" {
		return opts.Identifier
	}

	if opts.Title != "" {
		// todo: check title against regex?
		// go back to mapping strategy?
		return opts.Title
	}

	return time.Now().UTC().Format("2006-01-02T150405.999999999Z0700")
}

func uriEncode(s string) string {
	// QueryEscape replaces spaces with `+` which aren't unescaped by archive dot org
	// PathEscape seems to be compatible with archive dot org
	return fmt.Sprintf("uri(%s)", url.PathEscape(s))
}
