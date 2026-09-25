package sumex

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// request is one perp-api call. path is the route below the host ("/v1/orders"), with any path
// parameter already escaped by the caller; query and body are encoded once, here, and the exact
// encoded bytes are both sent and signed.
type request struct {
	method string
	path   string
	query  url.Values
	body   interface{}
	signed bool
}

// apiError carries a non-2xx perp-api response. perp-api errors are {message, code, details?};
// anything else occupying the channel (a proxy page, an HTML 502) is kept bounded so it cannot
// flood the caller's logs.
type apiError struct {
	StatusCode int
	Code       string
	Message    string
	Raw        []byte
}

func (e apiError) Error() string {
	if e.Code != "" || e.Message != "" {
		return fmt.Sprintf("sumex: HTTP %d %s: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("sumex: HTTP %d: %s", e.StatusCode, summarizeErrorBody(e.Raw))
}

const maxErrorBodyBytes = 512

func summarizeErrorBody(raw []byte) string {
	body := strings.Join(strings.Fields(string(raw)), " ")
	if body == "" {
		return "(empty response body)"
	}
	if len(body) > maxErrorBodyBytes {
		return body[:maxErrorBodyBytes] + "... (truncated)"
	}
	return body
}

func newAPIError(status int, raw []byte) apiError {
	e := apiError{StatusCode: status, Raw: raw}
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &body) == nil {
		e.Code = body.Code
		e.Message = body.Message
	}
	return e
}

// isAPIErrorCode reports whether err is a perp-api error with the given status and code.
func isAPIErrorCode(err error, status int, code string) bool {
	var e apiError
	return errors.As(err, &e) && e.StatusCode == status && e.Code == code
}

// ===============TRANSPORT=================

// No retries, deliberately — unlike utils.Request.DoFunc, which retries GETs on 429/5xx. perp-api
// records every accepted signature for its replay window BEFORE rate limiting or proxying, so a
// retry that resends the same signed bytes is refused as 401 REPLAYED_REQUEST and turns a transient
// failure into a misleading auth error. A caller that wants to retry calls the action again, which
// signs afresh.
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
	},
}

func (c *transport) httpClientFor() (*http.Client, error) {
	if c.Proxy == "" {
		return httpClient, nil
	}
	proxyURL, err := url.Parse(c.Proxy)
	if err != nil {
		return nil, fmt.Errorf("sumex: invalid proxy: %w", err)
	}
	return &http.Client{
		Timeout: httpClient.Timeout,
		Transport: &http.Transport{
			Proxy:               http.ProxyURL(proxyURL),
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}, nil
}

func (c *transport) callAPI(ctx context.Context, r *request) ([]byte, error) {
	if c.BaseURL == "" {
		return nil, errHostNotConfigured
	}

	requestURI := r.path
	if len(r.query) > 0 {
		requestURI += "?" + r.query.Encode()
	}

	var body []byte
	if r.body != nil {
		b, err := json.Marshal(r.body)
		if err != nil {
			return nil, err
		}
		body = b
	}

	req, err := http.NewRequestWithContext(ctx, r.method, c.BaseURL+requestURI, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	// perp-api only captures the raw body under application/json; any other content type makes it
	// verify the signature against "" and fail.
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	if r.signed {
		ts := nextTimestamp(c.apiKey, time.Now().UnixMilli()+c.TimeOffset)
		sig, err := signPayload(c.secretKey, ts, r.method, requestURI, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("x-timestamp", ts)
		req.Header.Set("x-signature", sig)
	}

	// Headers are not dumped: x-signature is replayable within the window.
	c.debug("%s %s%s\n", r.method, c.BaseURL, requestURI)
	c.debug("Body %s\n", string(body))

	client, err := c.httpClientFor()
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return nil, err
	}

	c.debug("response status code: %d\n", res.StatusCode)
	c.debug("response body: %s\n", string(data))

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, newAPIError(res.StatusCode, data)
	}
	return data, nil
}

// ===============SIGNING=================

// signPayload implements perp-api's user-key scheme (perp-api src/auth/api-key-auth.ts): an
// ed25519 signature over "timestamp\nMETHOD\nrequestURI\nbody", where requestURI is the path plus
// query exactly as sent and body the exact bytes sent ("" when there is none). The secret is the
// 32-byte ed25519 seed; key, secret and signature are all unpadded base64url.
func signPayload(secret, timestamp, method, requestURI string, body []byte) (string, error) {
	seed, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil || len(seed) != ed25519.SeedSize {
		return "", errors.New("sumex: the API secret is not a base64url-encoded 32-byte ed25519 seed")
	}
	msg := timestamp + "\n" + strings.ToUpper(method) + "\n" + requestURI + "\n" + string(body)
	sig := ed25519.Sign(ed25519.NewKeyFromSeed(seed), []byte(msg))
	return base64.RawURLEncoding.EncodeToString(sig), nil
}

// ed25519 is deterministic, so two byte-identical requests signed in the same millisecond carry the
// same signature and perp-api refuses the second as a replay — which is exactly what concurrent
// position polls look like. Handing each key a strictly increasing timestamp makes every signature
// distinct. It only holds within this process; perp-api's 30 s skew window absorbs the few
// milliseconds this can push a timestamp ahead of the clock.
var (
	timestampsMu sync.Mutex
	lastStamp    = map[string]int64{}

	pruneThreshold = timestampPruneAfter
)

const timestampPruneAfter = 10000

func nextTimestamp(apiKey string, now int64) string {
	timestampsMu.Lock()
	defer timestampsMu.Unlock()

	if last, ok := lastStamp[apiKey]; ok && now <= last {
		now = last + 1
	}
	lastStamp[apiKey] = now
	// Entries only matter within the replay window; sweep stale ones once the map has grown, and
	// move the threshold so a map of genuinely active keys is not swept on every call.
	if len(lastStamp) >= pruneThreshold {
		for k, v := range lastStamp {
			if now-v > 60_000 {
				delete(lastStamp, k)
			}
		}
		pruneThreshold = max(timestampPruneAfter, 2*len(lastStamp))
	}
	return strconv.FormatInt(now, 10)
}
