package web

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type (
	Client struct {
		Client *http.Client
	}

	SaveOptions struct {
		// SaveOutLinks indicates if we want the out-links saved.
		SaveOutLinks bool
		// SaveErrorPages indicates if we want to save error pages (HTTP Status=4xx,5xx)
		SaveErrorPages bool
		// SaveScreenShot indicates if we want to save a screen shot
		SaveScreenShot bool
		// SaveInMyWebArchive bool
		// EmailMeTheResults bool
	}

	Result struct {
		StatusCode int
		Headers    http.Header
	}
)

const (
	saveUrlFormat = "https://web.archive.org/save/%s"
)

func (c Client) Save(ctx context.Context, link *url.URL, options SaveOptions) (Result, error) {
	result := Result{}
	data := options.Values()
	data.Add("url", link.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf(saveUrlFormat, link.String()), strings.NewReader(data.Encode()))
	if err != nil {
		return result, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.Client.Do(req)
	if resp != nil {
		result.StatusCode = resp.StatusCode
		result.Headers = resp.Header
		if resp.Body != nil {
			defer resp.Body.Close()
		}
	}

	return result, err
}

func (options SaveOptions) Values() url.Values {
	values := url.Values{}
	if options.SaveOutLinks {
		values.Add("capture_outlinks", "on")
	}
	if options.SaveErrorPages {
		values.Add("capture_all", "on")
	}
	if options.SaveScreenShot {
		values.Add("capture_screenshot", "on")
	}
	return values
}
