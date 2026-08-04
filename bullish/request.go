package bullish

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/utils"
)

type aPIError struct {
	Code     string `json:"code"`
	Message  string `json:"msg"`
	Response []byte `json:"-"`
}

func (e aPIError) Error() string {
	// if e.IsValid() {
	// 	return fmt.Sprintf("<APIError> code=%s, msg=%s", e.Code, e.Message)
	// }
	return string(e.Response)
}

func (e aPIError) IsValid() bool {
	return e.Message == "" && e.Code == "0"
}

// ===============FUTURES=================

func (c *FuturesClient) callAPI(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error) {
	if c.Proxy != "" {
		r.Proxy = c.Proxy
	}
	r.BaseURL = c.BaseURL
	r.TimeOffset = c.TimeOffset
	r.TmpApi = c.apiKey
	r.TmpSig = c.secretKey
	r.TmpMemo = c.memo
	opts = append(opts, createFullURL, createBody, createSign, createHeaders, createReq)
	err = r.ParseRequest(opts...)
	if err != nil {
		return []byte{}, &http.Header{}, err

	}
	c.debug("FullURL %s\n", r.FullURL)
	c.debug("Body %s\n", r.BodyString)
	c.debug("Sign %s\n", r.Sign)
	c.debug("Headers %+v\n", r.Header)

	// return []byte{}, &http.Header{}, err

	req, err := http.NewRequest(r.Method, r.FullURL, r.Body)
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	req = req.WithContext(ctx)
	req.Header = r.Header
	c.debug("request: %#v\n", req)

	res, err := r.DoFunc(req)
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	data, err = r.ReadAllBody(res.Body, res.Header.Get("Content-Encoding"))
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	defer func() {
		cerr := res.Body.Close()

		if err == nil && cerr != nil {
			err = cerr
		}
	}()

	c.debug("response: %#v\n", res)
	c.debug("response body: %s\n", string(data))
	c.debug("response status code: %d\n", res.StatusCode)

	if res.StatusCode != 200 {
		return nil, &res.Header, errors.New(fmt.Sprintf("StatusCode: %d %s", res.StatusCode, string(data)))
		// apiErr := new(aPIError)
		// e := json.Unmarshal(data, apiErr)
		// if e != nil {
		// 	c.debug("failed to unmarshal json: %s\n", e)
		// }
		// if !apiErr.IsValid() {
		// 	c.debug("Answer Not Walid: %+v\n", apiErr)
		// 	apiErr.Response = data
		// 	return nil, &res.Header, apiErr
		// }
	}

	return data, &res.Header, nil
}

// =============END================
func createFullURL(r *utils.Request) error {
	fullURL := fmt.Sprintf("%s%s", r.BaseURL, r.Endpoint)
	queryString := r.Query.Encode()
	if queryString != "" {
		fullURL = fmt.Sprintf("%s?%s", fullURL, queryString)
	}
	r.QueryString = queryString
	r.FullURL = fullURL
	return nil
}

func createBody(r *utils.Request) error {
	body := &bytes.Buffer{}
	j, err := json.Marshal(r.Form)
	if err != nil {
		return err
	}
	bodyString := string(j)
	if bodyString == "{}" {
		bodyString = ""
	} else {
		bodyString = strings.Replace(bodyString, "[\"", "\"", -1)
		bodyString = strings.Replace(bodyString, "\"]", "\"", -1)
	}

	if bodyString != "" {
		body = bytes.NewBufferString(bodyString)
	}
	r.BodyString = bodyString
	r.Body = body
	return nil
}

func createSign(r *utils.Request) error {
	if r.SecType == utils.SecTypeSigned {
		r.Timestamp = utils.CurrentTimestamp() - r.TimeOffset

		// "/trading-api/v1/users/hmac/login"
		sf, err := utils.SignFunc(utils.KeyTypeHmac)
		if err != nil {
			return err
		}

		path := r.Endpoint
		// if r.QueryString != "" {
		// 	path = fmt.Sprintf("%s?%s", path, r.QueryString)
		// }
		raw := fmt.Sprintf("%d%d%s%s", r.Timestamp, r.Timestamp*1000, r.Method, path)

		if r.Method == http.MethodPost {
			raw = fmt.Sprintf("%d%d%s%s%s", r.Timestamp, r.Timestamp*1000, r.Method, path, r.BodyString)
		}

		if r.Endpoint != "/trading-api/v1/users/hmac/login" {
			raw = utils.GetSHA256Hash(raw)
		}

		sign, err := sf(r.TmpSig, raw)
		if err != nil {
			return err
		}
		r.TmpSig = ""
		r.Sign = *sign
	}
	return nil
}

func createHeaders(r *utils.Request) error {
	header := http.Header{}
	if r.Header != nil {
		header = r.Header.Clone()
	}

	header.Set("Content-Type", "application/json")
	if r.SecType == utils.SecTypeSigned {
		header.Set("BX-TIMESTAMP", utils.Int64ToString(r.Timestamp))
		header.Set("BX-NONCE", utils.Int64ToString(r.Timestamp*1000))
		header.Set("BX-PUBLIC-KEY", r.TmpApi)
		header.Set("BX-SIGNATURE", r.Sign)
		if r.Endpoint != "/trading-api/v1/users/hmac/login" {
			header.Set("Authorization", fmt.Sprintf("Bearer %s", r.TmpMemo))
		}
	}
	r.TmpApi = ""
	r.TmpMemo = ""
	r.Header = header
	return nil
}

func createReq(r *utils.Request) error {
	return nil
}
