package gateio

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/utils"
)

type aPIError struct {
	Code     string `json:"label"`
	Message  string `json:"message"`
	Response []byte `json:"-"`
}

func (e aPIError) Error() string {
	// if e.IsValid() {
	// 	return fmt.Sprintf("<APIError> label=%s, msg=%s", e.Code, e.Message)
	// }
	return string(e.Response)
}

func (e aPIError) IsValid() bool {
	return e.Message == "" && e.Code == ""
}

// ===============SPOT=================
func (c *SpotClient) callAPI(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error) {
	if c.Proxy != "" {
		r.Proxy = c.Proxy
	}
	if c.BrokerID != "" {
		header := http.Header{}
		if r.Header != nil {
			header = r.Header.Clone()
		}
		header.Set("X-Gate-Channel-Id", c.BrokerID)
		r.Header = header
	}
	r.BaseURL = c.BaseURL
	r.TimeOffset = c.TimeOffset
	r.TmpApi = c.apiKey
	r.TmpSig = c.secretKey

	opts = append(opts, createFullURL, createBody, createSign, createHeadersSpot, createReq)
	err = r.ParseRequest(opts...)
	if err != nil {
		return []byte{}, &http.Header{}, err

	}

	c.debug("FullURL %s\n", r.FullURL)
	c.debug("Body %s\n", r.BodyString)
	c.debug("Sign %s\n", r.Sign)
	c.debug("Headers %+v\n", r.Header)

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

	if res.StatusCode >= http.StatusBadRequest {
		apiErr := new(aPIError)
		e := json.Unmarshal(data, apiErr)
		if e != nil {
			c.debug("failed to unmarshal json: %s\n", e)
		}
		if !apiErr.IsValid() {
			c.debug("Answer Not Walid: %+v\n", apiErr)
			apiErr.Response = data
			return nil, &res.Header, apiErr
		}
	}
	return data, &res.Header, nil
}

// ===============FUTURES=================

func (c *FuturesClient) callAPI(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error) {
	if c.Proxy != "" {
		r.Proxy = c.Proxy
	}
	if c.BrokerID != "" {
		header := http.Header{}
		if r.Header != nil {
			header = r.Header.Clone()
		}
		header.Set("X-Gate-Channel-Id", c.BrokerID)
		r.Header = header
	}

	r.BaseURL = c.BaseURL
	r.TimeOffset = c.TimeOffset
	r.TmpApi = c.apiKey
	r.TmpSig = c.secretKey

	opts = append(opts, createFullURL, createBody, createSign, createHeaders, createReq)
	err = r.ParseRequest(opts...)
	if err != nil {
		return []byte{}, &http.Header{}, err

	}

	c.debug("FullURL %s\n", r.FullURL)
	c.debug("Body %s\n", r.BodyString)
	c.debug("Sign %s\n", r.Sign)
	c.debug("Headers %+v\n", r.Header)

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

	if res.StatusCode >= http.StatusBadRequest {
		apiErr := new(aPIError)
		e := json.Unmarshal(data, apiErr)
		if e != nil {
			c.debug("failed to unmarshal json: %s\n", e)
		}
		if !apiErr.IsValid() {
			c.debug("Answer Not Walid: %+v\n", apiErr)
			apiErr.Response = data
			return nil, &res.Header, apiErr
		}
	}
	return data, &res.Header, nil
}

// =============END================

func createFullURL(r *utils.Request) error {
	fullURL := fmt.Sprintf("%s%s", r.BaseURL, r.Endpoint)
	r.Timestamp = utils.CurrentSecondsTimestamp() - r.TimeOffset
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

		if r.Form.Get("is_batch") != "" {
			bodyString = r.Form.Get("is_batch")
		} else {
			bodyString = strings.Replace(bodyString, "[\"", "\"", -1)
			bodyString = strings.Replace(bodyString, "\"]", "\"", -1)

			if r.Form.Get("attachAlgoOrds") != "" {
				bodyString = strings.Replace(bodyString, "\"[", "[", -1)
				bodyString = strings.Replace(bodyString, "]\"", "]", -1)
				bodyString = strings.Replace(bodyString, `\"`, `"`, -1)
			}

			if r.Form.Get("initial") != "" && r.Form.Get("trigger") != "" {
				bodyString = strings.Replace(bodyString, `"{`, `{`, -1)
				bodyString = strings.Replace(bodyString, `"}`, `}`, -1)
				bodyString = strings.Replace(bodyString, `\}`, `"}`, -1)
				bodyString = strings.Replace(bodyString, `\"`, `"`, -1)
				bodyString = strings.Replace(bodyString, `}"`, `}`, -1)

			}
		}
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
		sf, err := utils.SignFunc(utils.KeyTypeHmacHex512)
		if err != nil {
			return err
		}
		h := sha512.New()
		if r.BodyString != "" {
			h.Write([]byte(r.BodyString))
		}
		hashedPayload := hex.EncodeToString(h.Sum(nil))
		raw := fmt.Sprintf("%s\n%s\n%s\n%s\n%d", r.Method, r.Endpoint, r.QueryString, hashedPayload, r.Timestamp)
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

	header.Set("Accept", "application/json")
	header.Set("Content-Type", "application/json")
	if r.SecType == utils.SecTypeSigned {
		header.Set("KEY", r.TmpApi)
		header.Set("SIGN", r.Sign)
		header.Set("Timestamp", fmt.Sprintf("%d", r.Timestamp))
	}
	r.TmpApi = ""
	r.Header = header
	return nil
}

func createHeadersSpot(r *utils.Request) error {
	header := http.Header{}
	if r.Header != nil {
		header = r.Header.Clone()
	}

	header.Set("Accept", "application/json")
	header.Set("Content-Type", "application/json")
	if r.SecType == utils.SecTypeSigned {
		header.Set("KEY", r.TmpApi)
		header.Set("SIGN", r.Sign)
		header.Set("Timestamp", fmt.Sprintf("%d", r.Timestamp))
	}
	r.TmpApi = ""
	r.Header = header
	return nil
}

func createReq(r *utils.Request) error {
	return nil
}
