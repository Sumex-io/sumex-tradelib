package blofin

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Betarost/onetrades/utils"
)

// ---------------------------
// Blofin error format
// ---------------------------
type apiError struct {
	Code     string          `json:"code"`
	Message  string          `json:"msg"`
	Response json.RawMessage `json:"-"`
}

func (e apiError) Error() string {
	return string(e.Response)

}

func (e apiError) IsValid() bool {
	return e.Code == "0"
}

// =================== SPOT ===================
func (c *SpotClient) callAPI(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) ([]byte, *http.Header, error) {

	if c.Proxy != "" {
		r.Proxy = c.Proxy
	}

	r.BaseURL = c.BaseURL
	r.TimeOffset = c.TimeOffset

	r.TmpApi = c.apiKey
	r.TmpSig = c.secretKey
	r.TmpMemo = c.memo

	// Build request
	opts = append(opts, createFullURL, createBody, createSignBlofin, createHeadersBlofin, createReq)
	err := r.ParseRequest(opts...)
	if err != nil {
		return nil, &http.Header{}, err
	}

	c.debug("URL: %s\n", r.FullURL)
	c.debug("Body: %s\n", r.BodyString)
	c.debug("Sign: %s\n", r.Sign)
	c.debug("Headers: %+v\n", r.Header)

	req, err := http.NewRequest(r.Method, r.FullURL, r.Body)
	if err != nil {
		return nil, &http.Header{}, err
	}

	req = req.WithContext(ctx)
	req.Header = r.Header

	res, err := r.DoFunc(req)
	if err != nil {
		return nil, &http.Header{}, err
	}

	data, err := r.ReadAllBody(res.Body, res.Header.Get("Content-Encoding"))
	if err != nil {
		return nil, &http.Header{}, err
	}
	defer res.Body.Close()

	c.debug("Response code: %d\n", res.StatusCode)
	c.debug("Response body: %s\n", string(data))

	apiErr := apiError{}
	_ = json.Unmarshal(data, &apiErr)

	if !apiErr.IsValid() {
		apiErr.Response = data
		return nil, &res.Header, apiErr
	}

	return data, &res.Header, nil
}

// ================= FUTURES ==================
func (c *FuturesClient) callAPI(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) ([]byte, *http.Header, error) {

	if c.Proxy != "" {
		r.Proxy = c.Proxy
	}

	r.BaseURL = c.BaseURL
	r.TimeOffset = c.TimeOffset

	r.TmpApi = c.apiKey
	r.TmpSig = c.secretKey
	r.TmpMemo = c.memo

	opts = append(opts, createFullURL, createBody, createSignBlofin, createHeadersBlofin, createReq)
	err := r.ParseRequest(opts...)
	if err != nil {
		return nil, &http.Header{}, err
	}

	c.debug("URL: %s\n", r.FullURL)
	c.debug("Body: %s\n", r.BodyString)
	c.debug("Sign: %s\n", r.Sign)
	c.debug("Headers: %+v\n", r.Header)

	req, err := http.NewRequest(r.Method, r.FullURL, r.Body)
	if err != nil {
		return nil, &http.Header{}, err
	}

	req = req.WithContext(ctx)
	req.Header = r.Header

	res, err := r.DoFunc(req)
	if err != nil {
		return nil, &http.Header{}, err
	}

	data, err := r.ReadAllBody(res.Body, res.Header.Get("Content-Encoding"))
	if err != nil {
		return nil, &http.Header{}, err
	}
	defer res.Body.Close()

	c.debug("Response code: %d\n", res.StatusCode)
	c.debug("Response body: %s\n", string(data))

	apiErr := apiError{}
	_ = json.Unmarshal(data, &apiErr)

	if !apiErr.IsValid() {
		apiErr.Response = data
		return nil, &res.Header, apiErr
	}

	return data, &res.Header, nil
}

// =================== COMMON BUILDERS ===================

// Build Full URL
func createFullURL(r *utils.Request) error {
	fullURL := fmt.Sprintf("%s%s", r.BaseURL, r.Endpoint)
	if q := r.Query.Encode(); q != "" {
		fullURL = fullURL + "?" + q
	}
	r.FullURL = fullURL
	return nil
}

func createBody(r *utils.Request) error {
	// Для GET тела нет
	if r.Method == http.MethodGet {
		r.Body = bytes.NewBuffer(nil)
		r.BodyString = ""
		return nil
	}

	if len(r.Form) == 0 {
		r.Body = bytes.NewBuffer(nil)
		r.BodyString = ""
		return nil
	}

	bodyMap := make(map[string]interface{}, len(r.Form))
	for k, vs := range r.Form {
		if len(vs) == 1 {
			bodyMap[k] = vs[0]
		} else {
			bodyMap[k] = vs
		}
	}

	j, err := json.Marshal(bodyMap)
	if err != nil {
		return err
	}

	r.BodyString = string(j)
	r.Body = bytes.NewBuffer(j)
	return nil
}

func createSignBlofin(r *utils.Request) error {
	if r.SecType != utils.SecTypeSigned {
		return nil
	}

	// 1) timestamp в миллисекундах
	r.Timestamp = utils.CurrentTimestamp() - r.TimeOffset
	ts := fmt.Sprintf("%d", r.Timestamp)

	// 2) nonce = timestamp (так делает ccxt, и это ок по докам)
	nonce := ts

	// 3) requestPath (+ query для GET)
	requestPath := r.Endpoint
	queryStr := r.Query.Encode()

	signBody := ""

	if r.Method == http.MethodGet {
		if queryStr != "" {
			requestPath = requestPath + "?" + queryStr
		}
	} else {
		if r.BodyString != "" {
			signBody = r.BodyString
		}
	}

	// 4) prehash: path + method + timestamp + nonce + body
	auth := requestPath + r.Method + ts + nonce + signBody

	// 5) HMAC-SHA256 -> hex
	sfHex, err := utils.SignFunc(utils.KeyTypeHmac)
	if err != nil {
		return err
	}
	hexStr, err := sfHex(r.TmpSig, auth)
	if err != nil {
		return err
	}

	// 6) Base64 от hex-строки
	r.Sign = base64.StdEncoding.EncodeToString([]byte(*hexStr))

	return nil
}

func createHeadersBlofin(r *utils.Request) error {
	header := http.Header{}

	// Как в ccxt: Content-Type ставят только когда реально есть body
	if r.SecType == utils.SecTypeSigned && r.Method != http.MethodGet && r.BodyString != "" {
		header.Set("Content-Type", "application/json")
	} else {
		// Можно и так оставить, если у тебя везде так принято:
		header.Set("Content-Type", "application/json")
	}

	if r.SecType == utils.SecTypeSigned {
		tsStr := fmt.Sprintf("%d", r.Timestamp)

		header.Set("ACCESS-KEY", r.TmpApi)
		header.Set("ACCESS-PASSPHRASE", r.TmpMemo)
		header.Set("ACCESS-TIMESTAMP", tsStr)
		header.Set("ACCESS-NONCE", tsStr)
		header.Set("ACCESS-SIGN", r.Sign)
	}

	r.Header = header
	return nil
}

func createReq(r *utils.Request) error {
	return nil
}
