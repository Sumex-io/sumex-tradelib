package utils

import (
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type secType int

const (
	SecTypeNone secType = iota
	SecTypeAPIKey
	SecTypeSigned
)

type Params map[string]interface{}

type Request struct {
	Method      string
	BaseURL     string
	Endpoint    string
	Proxy       string
	BrokerID    string
	TimeOffset  int64
	Query       url.Values
	QueryString string
	Form        url.Values
	RecvWindow  int64
	Timestamp   int64
	SecType     secType
	Sign        string
	Header      http.Header
	Body        io.Reader
	BodyString  string
	FullURL     string
	TmpSig      string
	TmpApi      string
	TmpMemo     string
}

type RequestOption func(*Request) error

// -------- HTTP client/transport (централизовано) --------

var defaultTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	MaxIdleConns:          200,
	MaxIdleConnsPerHost:   50,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   5 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
}

var defaultHTTPClient = &http.Client{
	Timeout:   6 * time.Second,
	Transport: defaultTransport,
}

// Возможность переопределить клиент
func SetHTTPClient(c *http.Client) {
	if c != nil {
		defaultHTTPClient = c
	}
}

// Безопасное «клонирование» транспорта без копирования внутренних mutex.
func cloneDefaultTransportWithProxy(proxyURL *url.URL) *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyURL(proxyURL),
		DialContext:           defaultTransport.DialContext,
		MaxIdleConns:          defaultTransport.MaxIdleConns,
		MaxIdleConnsPerHost:   defaultTransport.MaxIdleConnsPerHost,
		IdleConnTimeout:       defaultTransport.IdleConnTimeout,
		TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
		ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
		TLSClientConfig:       defaultTransport.TLSClientConfig,
		ProxyConnectHeader:    defaultTransport.ProxyConnectHeader,
		ForceAttemptHTTP2:     defaultTransport.ForceAttemptHTTP2,
	}
}

// -------- Request helpers --------

func (r *Request) SetParam(key string, value interface{}) *Request {
	if r.Query == nil {
		r.Query = url.Values{}
	}
	r.Query.Set(key, fmt.Sprintf("%v", value))
	return r
}

func (r *Request) SetParams(m Params) *Request {
	for k, v := range m {
		r.SetParam(k, v)
	}
	return r
}

func (r *Request) SetFormParam(key string, value interface{}) *Request {
	if r.Form == nil {
		r.Form = url.Values{}
	}
	r.Form.Set(key, fmt.Sprintf("%v", value))
	return r
}

func (r *Request) SetFormParams(m Params) *Request {
	for k, v := range m {
		r.SetFormParam(k, v)
	}
	return r
}

func (r *Request) SetFormParamStruct(key string, value interface{}) *Request {
	if r.Form == nil {
		r.Form = url.Values{}
	}

	v := string(value.([]byte))
	// v = strings.Replace(v, `\"`, `"`, -1)
	// v = strings.Replace(v, `"{`, `{`, -1)
	// v = strings.Replace(v, `"}`, `}`, -1)

	r.Form.Set(key, v)
	return r
}

func (r *Request) SetFormParamsStruct(m Params) *Request {
	for k, v := range m {
		r.SetFormParamStruct(k, v)
	}
	return r
}

func (r *Request) validate() error {
	if r.Query == nil {
		r.Query = url.Values{}
	}
	if r.Form == nil {
		r.Form = url.Values{}
	}
	return nil
}

func (r *Request) ParseRequest(opts ...RequestOption) error {
	if err := r.validate(); err != nil {
		return err
	}
	for _, opt := range opts {
		if e := opt(r); e != nil {
			return e
		}
	}
	return nil
}

func (c *Request) ReadAllBody(r io.Reader, content string) ([]byte, error) {
	switch strings.ToLower(content) {
	case "gzip":
		reader, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	default:
		return io.ReadAll(r)
	}
}

// -------- Retry helpers (только для безопасных запросов) --------

func shouldRetryStatus(code int) bool {
	return code == 408 || code == 429 || code >= 500
}

func isNetErr(err error) bool {

	if ne, ok := err.(net.Error); ok {
		return ne.Timeout() || ne.Temporary()
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "timeout") ||
		strings.Contains(s, "connection") ||
		strings.Contains(s, "reset") ||
		strings.Contains(s, "refused") ||
		strings.Contains(s, "closed")
}

func parseRetryAfter(h string) time.Duration {
	h = strings.TrimSpace(h)
	if h == "" {
		return 0
	}
	if n, err := strconv.Atoi(h); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

func fullJitter(base, max time.Duration, attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	ceil := base << (attempt - 1)
	if ceil > max {
		ceil = max
	}
	if ceil <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(ceil) + 1))
}

// doWithClient — общая обвязка запроса + ретраи для безопасных методов.
// Ретраим ТОЛЬКО GET (HEAD по желанию), POST — не ретраим (без идемпотентности).
func doWithClient(client *http.Client, req *http.Request) (*http.Response, error) {
	// гарантируем gzip по умолчанию
	if req.Header.Get("Accept-Encoding") == "" {
		req.Header.Set("Accept-Encoding", "gzip")
	}

	method := strings.ToUpper(req.Method)
	retryable := method == http.MethodGet // только GET

	retryMax := 3
	baseDelay := 200 * time.Millisecond
	maxDelay := 2 * time.Second

	attempts := 1
	if retryable {
		attempts = retryMax + 1
	}

	var lastNetErr error
	for a := 1; a <= attempts; a++ {

		resp, err := client.Do(req)
		if err != nil {
			if retryable && a < attempts && isNetErr(err) {
				lastNetErr = err
				time.Sleep(fullJitter(baseDelay, maxDelay, a))
				continue
			}
			return nil, err
		}

		// Success
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		if retryable && a < attempts && shouldRetryStatus(resp.StatusCode) {
			delay := fullJitter(baseDelay, maxDelay, a)
			if ra := parseRetryAfter(resp.Header.Get("Retry-After")); ra > 0 {
				delay = ra
			}

			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			time.Sleep(delay)
			continue
		}

		// Неретрайная ошибка или последняя попытка — возвращаем как есть
		return resp, nil
	}

	// Дошли сюда только если всё закончилось сетевой ошибкой на всех попытках
	if lastNetErr != nil {
		return nil, lastNetErr
	}
	return nil, fmt.Errorf("http retry exhausted")
}

// DoFunc — единая точка отправки запроса.
func (c *Request) DoFunc(req *http.Request) (*http.Response, error) {

	if c.Proxy != "" {
		proxyURL, err := url.Parse(c.Proxy)
		if err != nil {
			log.Println("DoFunc: invalid proxy:", err)
		} else {
			client := &http.Client{
				Timeout:   6 * time.Second,
				Transport: cloneDefaultTransportWithProxy(proxyURL),
			}
			return doWithClient(client, req)
		}
	}

	return doWithClient(defaultHTTPClient, req)
}
