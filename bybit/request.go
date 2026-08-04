package bybit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/utils"
)

type aPIError struct {
	Code     int64  `json:"retCode"`
	Message  string `json:"retMsg"`
	Response []byte `json:"-"`
}

func (e aPIError) Error() string {
	// if e.IsValid() {
	// 	return fmt.Sprintf("<APIError> code=%d, msg=%s", e.Code, e.Message)
	// }
	return string(e.Response)
}

func (e aPIError) IsValid() bool {
	return e.Code == 0
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
		header.Set("Referer", c.BrokerID)
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
		header.Set("Referer", c.BrokerID)
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
		r.Timestamp = utils.CurrentTimestamp() - r.TimeOffset

		sf, err := utils.SignFunc(utils.KeyTypeHmac)
		if err != nil {
			return err
		}
		raw := fmt.Sprintf("%d%s5000%s", r.Timestamp, r.TmpApi, r.QueryString)

		if r.BodyString != "" {
			raw = fmt.Sprintf("%d%s5000%s", r.Timestamp, r.TmpApi, r.BodyString)
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
	if r.BodyString != "" {
		header.Set("Content-Type", "application/json")
	}
	if r.SecType == utils.SecTypeSigned {
		header.Set("X-BAPI-API-KEY", r.TmpApi)
		header.Set("X-BAPI-SIGN", r.Sign)
		header.Set("X-BAPI-TIMESTAMP", fmt.Sprintf("%d", r.Timestamp))
		header.Set("X-BAPI-RECV-WINDOW", "5000")
	}
	r.TmpApi = ""
	r.Header = header
	return nil
}

func createReq(r *utils.Request) error {
	return nil
}
