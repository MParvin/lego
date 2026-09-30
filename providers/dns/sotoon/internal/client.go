package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-acme/lego/v5/internal/errutils"
	"github.com/go-acme/lego/v5/internal/useragent"
)

// defaultBaseURL the default API endpoint.
const defaultBaseURL = "https://api.sotoon.ir/delivery/v2.1/global"

const mergePatchContentType = "application/merge-patch+json"

// Client the Sotoon API client.
type Client struct {
	token         string
	workspaceUUID string

	BaseURL    *url.URL
	HTTPClient *http.Client
}

// NewClient creates a new Client.
func NewClient(token, workspaceUUID string) (*Client, error) {
	if token == "" || workspaceUUID == "" {
		return nil, errors.New("credentials missing")
	}

	baseURL, _ := url.Parse(defaultBaseURL)

	return &Client{
		token:         token,
		workspaceUUID: workspaceUUID,
		BaseURL:       baseURL,
		HTTPClient:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// ListDomainZones lists all DomainZone resources in the workspace.
// https://docs.sotoon.ir/products/cdn/api-reference/api-getting-started
func (c *Client) ListDomainZones(ctx context.Context) ([]DomainZone, error) {
	endpoint := c.BaseURL.JoinPath("workspaces", c.workspaceUUID, "domainzones")

	req, err := newJSONRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	result := new(DomainZoneList)

	err = c.do(req, result)
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}

// GetDomainZone gets a DomainZone by metadata.name.
// https://docs.sotoon.ir/products/cdn/api-reference/api-getting-started
func (c *Client) GetDomainZone(ctx context.Context, name string) (*DomainZone, error) {
	endpoint := c.BaseURL.JoinPath("workspaces", c.workspaceUUID, "domainzones", name)

	req, err := newJSONRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	result := new(DomainZone)

	err = c.do(req, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// PatchDomainZone updates a DomainZone using JSON merge-patch.
// https://docs.sotoon.ir/products/cdn/api-reference/api-getting-started
func (c *Client) PatchDomainZone(ctx context.Context, name string, patch DomainZonePatch) (*DomainZone, error) {
	endpoint := c.BaseURL.JoinPath("workspaces", c.workspaceUUID, "domainzones", name)

	req, err := newJSONRequest(ctx, http.MethodPatch, endpoint, patch)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", mergePatchContentType)

	result := new(DomainZone)

	err = c.do(req, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) do(req *http.Request, result any) error {
	useragent.SetHeader(req.Header)

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return errutils.NewHTTPDoError(req, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode/100 != 2 {
		return parseError(req, resp)
	}

	if result == nil {
		return nil
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return errutils.NewReadResponseError(req, resp.StatusCode, err)
	}

	err = json.Unmarshal(raw, result)
	if err != nil {
		return errutils.NewUnmarshalError(req, resp.StatusCode, raw, err)
	}

	return nil
}

func newJSONRequest(ctx context.Context, method string, endpoint *url.URL, payload any) (*http.Request, error) {
	buf := new(bytes.Buffer)

	if payload != nil {
		err := json.NewEncoder(buf).Encode(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to create request JSON body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), buf)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func parseError(req *http.Request, resp *http.Response) error {
	raw, _ := io.ReadAll(resp.Body)

	var errAPI APIError

	err := json.Unmarshal(raw, &errAPI)
	if err != nil || errAPI.Message == "" {
		return errutils.NewUnexpectedStatusCodeError(req, resp.StatusCode, raw)
	}

	return &errAPI
}
