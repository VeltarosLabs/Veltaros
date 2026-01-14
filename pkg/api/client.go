package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

type Option func(*Client)

func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		if c != nil {
			cl.http = c
		}
	}
}

func New(baseURL string, opts ...Option) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, errors.New("baseURL must not be empty")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	cl := &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	for _, o := range opts {
		o(cl)
	}
	return cl, nil
}

func (c *Client) Health(ctx context.Context) (Health, error) {
	var out Health
	if err := c.getJSON(ctx, "/healthz", &out); err != nil {
		return Health{}, err
	}
	return out, nil
}

func (c *Client) Version(ctx context.Context) (VersionInfo, error) {
	var out VersionInfo
	if err := c.getJSON(ctx, "/version", &out); err != nil {
		return VersionInfo{}, err
	}
	return out, nil
}

func (c *Client) Status(ctx context.Context) (NodeStatus, error) {
	var out NodeStatus
	if err := c.getJSON(ctx, "/status", &out); err != nil {
		return NodeStatus{}, err
	}
	return out, nil
}

func (c *Client) Peers(ctx context.Context) (PeerList, error) {
	var out PeerList
	if err := c.getJSON(ctx, "/peers", &out); err != nil {
		return PeerList{}, err
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("http %s %s: status %d", http.MethodGet, path, resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}
