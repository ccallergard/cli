package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/exercism/cli/config"
	"github.com/exercism/cli/debug"
)

var (
	// UserAgent lets the API know where the call is being made from.
	// It's overridden from the root command so that we can set the version.
	UserAgent = "github.com/exercism/cli"

	// DefaultHTTPClient configures a timeout to use by default.
	DefaultHTTPClient = &http.Client{Timeout: 10 * time.Second}
)

// Client is an http client that is configured for Exercism.
type Client struct {
	*http.Client
	ContentType string
	Token       string
	BaseURL     string
}

// NewClient returns an Exercism API client.
func NewClient(token, baseURL string) (*Client, error) {
	return &Client{
		Client:  DefaultHTTPClient,
		Token:   token,
		BaseURL: baseURL,
	}, nil
}

// NewRequest returns an http.Request with information for the Exercism API.
func (c *Client) NewRequest(method, url string, body io.Reader) (*http.Request, error) {
	if c.Client == nil {
		c.Client = DefaultHTTPClient
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", UserAgent)
	if c.ContentType == "" {
		req.Header.Set("Content-Type", "application/json")
	} else {
		req.Header.Set("Content-Type", c.ContentType)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	}

	return req, nil
}

// Do performs an http.Request and optionally parses the response body into the given interface.
func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	debug.DumpRequest(req)

	res, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	debug.DumpResponse(res)

	switch res.StatusCode {
	case http.StatusNoContent:
		return res, nil
	case http.StatusInternalServerError:
		// TODO: if it's json, and it has an error key, print the message.
		return nil, fmt.Errorf("%s", res.Status)
	case http.StatusUnauthorized:
		siteURL := config.InferSiteURL(c.BaseURL)
		return nil, fmt.Errorf("unauthorized request. Please run the configure command. You can find your API token at %s/my/settings", siteURL)
	default:
		if v != nil {
			defer res.Body.Close()

			if err := json.NewDecoder(res.Body).Decode(v); err != nil {
				return nil, fmt.Errorf("unable to parse API response - %s", err)
			}
		}
	}

	return res, nil
}

// ValidateToken calls the API to determine whether the token is valid.
func (c *Client) ValidateToken() error {
	url := fmt.Sprintf("%s/validate_token", c.BaseURL)
	req, err := c.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	_, err = c.Do(req, nil)

	return err
}
