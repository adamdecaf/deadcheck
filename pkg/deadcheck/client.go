package deadcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

type Client interface {
	CheckIn(ctx context.Context, checkID string) (*CheckInResponse, error)
}

type Config struct {
	BaseAddress string
	HTTPClient  *http.Client
}

func NewClient(config Config) (Client, error) {
	_, err := url.Parse(config.BaseAddress)
	if err != nil {
		return nil, fmt.Errorf("parsing BaseAddress failed: %w", err)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &client{
		baseAddress: config.BaseAddress,
		httpClient:  httpClient,
	}, nil
}

type client struct {
	baseAddress string
	httpClient  *http.Client
}

type CheckInResponse struct {
	NextExpectedCheckIn time.Time `json:"nextExpectedCheckIn"`
}

// CheckIn updates the specified check's next expected alert time by extending it to the next scheduled interval.
// This function is typically called after an operation successfully completes. For example, after files are uploaded.
//
// Example usage:
//
//	response, err := client.CheckIn(ctx, "2pm-checkin")
//	if err != nil {
//	    log.Fatalf("Failed to check in: %v", err)
//	}
//	log.Printf("Check-in successful: next check-in expected by %v", response.NextExpectedCheckIn)
func (c *client) CheckIn(ctx context.Context, checkID string) (*CheckInResponse, error) {
	address, err := c.getAddress(fmt.Sprintf("/checks/%s/check-in", checkID))
	if err != nil {
		return nil, fmt.Errorf("getAddress for check-in: %w", err)
	}

	req, err := http.NewRequest("PUT", address, nil)
	if err != nil {
		return nil, fmt.Errorf("building check-in request: %w", err)
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("check-in failed: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	var response CheckInResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("decoding check-in response: %w", err)
	}
	return &response, nil
}

func (c *client) getAddress(after string) (string, error) {
	u, err := url.Parse(c.baseAddress)
	if err != nil {
		return "", fmt.Errorf("parsing baseAddress: %w", err)
	}
	u.Path = path.Join(u.Path, after)

	return u.String(), nil
}
