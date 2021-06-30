package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
		SubjectTags []string
		Creator     string
		Date        *time.Time
		Metadata    Metadata
		Collection  Collection

		AutoMakeBucket bool
		KeepOldVersion bool
		// TODO
		// Language string
		// License string
		// MediaType string
	}

	Metadata map[string]string

	Collection string
)

const (
	DefaultURL      = "https://s3.us.archive.org"
	uploadURLFormat = "%s/%s/%s"

	CollectionData   Collection = "opensource_media"
	CollectionMovies Collection = "opensource_movies"
	// TODO: other collections
)

func (c Client) Upload(ctx context.Context, opts UploadOptions) (interface{}, error) {
	// TODO: read https://archive.org/services/docs/api/ias3.html

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.uploadURL(opts), opts.Upload)
	if err != nil {
		return nil, err
	}

	for k, v := range opts.Metadata {
		if !strings.HasPrefix(k, "x-amz-meta") {
			k = "x-amz-meta-" + k
		}
		req.Header.Set(k, v)
	}

	req.Header.Set("authorization", fmt.Sprintf("LOW %s:%s", c.API.Key, c.API.Secret))
	if opts.Collection == "" {
		opts.Collection = CollectionData
	}
	req.Header.Set("x-archive-meta01-collection", string(opts.Collection))
	if opts.Title != "" {
		req.Header.Set("x-archive-meta-title", opts.Title)
	}
	if opts.AutoMakeBucket {
		req.Header.Set("x-amz-auto-make-bucket", "1")
	}
	if opts.KeepOldVersion {
		req.Header.Set("x-archive-keep-old-version", "1")
	}

	//TODO: --header 'x-archive-meta-mediatype:images'      --header 'x-archive-meta-language:eng' --header ':1'

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// TODO: Handle response
	//  - Slow down
	//  - Anything of value in the body?

	return nil, nil
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
