package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

// client groups HTTP access to a single BMC.
// It keeps connections alive and sleeps a random min-max duration between requests.
type client struct {
	base     string
	authHdr  string
	hc       *http.Client
	minSleep time.Duration
	maxSleep time.Duration
}

func newClient(base, user, pass string, timeout, minSleep, maxSleep time.Duration) *client {
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        1,
		MaxIdleConnsPerHost: 1,
		MaxConnsPerHost:     1, // physically cap concurrent connections at 1
		DisableKeepAlives:   false,
	}
	token := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	return &client{
		base:     base,
		authHdr:  "Basic " + token,
		hc:       &http.Client{Transport: tr, Timeout: timeout},
		minSleep: minSleep,
		maxSleep: maxSleep,
	}
}

// sleep waits a random duration between min and max before the next request.
func (c *client) sleep(ctx context.Context) {
	d := c.minSleep
	if c.maxSleep > c.minSleep {
		d += time.Duration(rand.Int63n(int64(c.maxSleep - c.minSleep)))
	}
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

// get fetches path (for example "/redfish/v1/") and returns the raw response body.
func (c *client) get(ctx context.Context, path string) ([]byte, error) {
	url := c.base + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHdr)
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("status %d", resp.StatusCode)
	}
	return body, nil
}
