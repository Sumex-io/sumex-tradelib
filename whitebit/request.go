package whitebit

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
// WhiteBIT error format
// ---------------------------
//
// У WhiteBIT в случае успеха чаще всего НЕТ поля code/message,
// а в случае ошибки возвращается объект вида:
//
//	{ "code": 113, "message": "Validation failed", "errors": {...} }
//
// Поэтому: "валидно" = code == 0 и message == "".
type apiError struct {
	Code     int             `json:"code"`
	Message  string          `json:"message"`
	Response json.RawMessage `json:"-"`
}

func (e apiError) Error() string {
	if len(e.Response) > 0 {
		return string(e.Response)
	}
	return fmt.Sprintf("WhiteBIT api error: code=%d, message=%s", e.Code, e.Message)
}

func (e apiError) IsValid() bool {
	return e.Code == 0 && e.Message == ""
}

// =================== SPOT ===================

func (c *SpotClient) callAPI(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) ([]byte, *http.Header, error) {
	if c.Proxy != "" {
		r.Proxy = c.Proxy
	}

	r.BaseURL = c.BaseURL
	r.TimeOffset = c.TimeOffset

	// временные поля для подписи
	r.TmpApi = c.apiKey
	r.TmpSig = c.secretKey
	r.TmpMemo = c.memo // для WhiteBIT не нужен, но пусть будет заполнен

	// WhiteBIT: своя цепочка билдеров
	opts = append(opts,
		createFullURLWhitebit,
		createBodyWhitebit,
		createSignWhitebit,
		createHeadersWhitebit,
		createReq,
	)

	if err := r.ParseRequest(opts...); err != nil {
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

	opts = append(opts,
		createFullURLWhitebit,
		createBodyWhitebit,
		createSignWhitebit,
		createHeadersWhitebit,
		createReq,
	)

	if err := r.ParseRequest(opts...); err != nil {
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

// WhiteBIT: BaseURL + Endpoint
// Для приватных запросов query-строка не используется (всё в body),
// для публичных — можно использовать r.Query как обычно.
func createFullURLWhitebit(r *utils.Request) error {
	fullURL := fmt.Sprintf("%s%s", r.BaseURL, r.Endpoint)

	// Для приватных запросов (SecTypeSigned) по доке всё должно идти в body.
	if r.SecType != utils.SecTypeSigned {
		if q := r.Query.Encode(); q != "" {
			fullURL = fullURL + "?" + q
		}
	}

	r.FullURL = fullURL
	return nil
}

// Формирование body.
// ПОВЕДЕНИЕ:
//
// - Публичные запросы (SecTypeNone):
//   - если Form пустой -> пустое тело
//   - если есть Form -> просто json(Form) как и на других биржах
//
// - Приватные запросы (SecTypeSigned, WhiteBIT V4/V1):
//   - формируем объект:
//     {
//     "request": "/api/v4/...",
//     "nonce":   <timestamp ms>,
//     ...прочие поля из r.Form
//     }
//   - r.Timestamp используем как nonce (ms).
func createBodyWhitebit(r *utils.Request) error {
	// PRIVATE (SIGNED)
	if r.SecType == utils.SecTypeSigned {
		// nonce = Unix ms с учётом possible TimeOffset
		if r.Timestamp == 0 {
			r.Timestamp = utils.CurrentTimestamp() - r.TimeOffset
		}

		payload := make(map[string]interface{})

		// обязательные поля по доке
		payload["request"] = r.Endpoint
		payload["nonce"] = r.Timestamp

		// дополнительные параметры из r.Form (SetFormParams)
		if r.Form != nil {
			for k, v := range r.Form {
				if len(v) == 0 {
					continue
				}
				// Берём первый элемент — у нас всё равно строки
				payload[k] = v[0]
			}
		}

		j, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		r.BodyString = string(j)
		r.Body = bytes.NewBuffer(j)
		return nil
	}

	// PUBLIC (без подписи) — максимально простой вариант
	if r.Form == nil || len(r.Form) == 0 {
		r.BodyString = ""
		r.Body = bytes.NewBuffer(nil)
		return nil
	}

	j, err := json.Marshal(r.Form)
	if err != nil {
		return err
	}

	r.BodyString = string(j)
	r.Body = bytes.NewBuffer(j)
	return nil
}

// Подпись WhiteBIT:
//
//	payload = base64(BodyString)
//	signature = hex(HMAC_SHA512(payload, key=api_secret))
func createSignWhitebit(r *utils.Request) error {
	if r.SecType != utils.SecTypeSigned {
		return nil
	}

	// base64 от JSON-body
	payload := base64.StdEncoding.EncodeToString([]byte(r.BodyString))

	sf, err := utils.SignFunc(utils.KeyTypeHmacHex512)
	if err != nil {
		return err
	}

	sign, err := sf(r.TmpSig, payload)
	if err != nil {
		return err
	}

	r.Sign = *sign
	return nil
}

// Заголовки WhiteBIT:
//
//	Content-Type: application/json
//	X-TXC-APIKEY:    apiKey
//	X-TXC-PAYLOAD:   base64(JSON body)
//	X-TXC-SIGNATURE: hex(HMAC_SHA512(payload))
func createHeadersWhitebit(r *utils.Request) error {
	header := http.Header{}
	if r.Header != nil {
		header = r.Header.Clone()
	}

	header.Set("Content-Type", "application/json")

	if r.SecType == utils.SecTypeSigned {
		payload := base64.StdEncoding.EncodeToString([]byte(r.BodyString))
		header.Set("X-TXC-APIKEY", r.TmpApi)
		header.Set("X-TXC-PAYLOAD", payload)
		header.Set("X-TXC-SIGNATURE", r.Sign)

		// подчистим временные поля
		r.TmpApi = ""
		r.TmpSig = ""
		r.TmpMemo = ""
	}

	r.Header = header
	return nil
}

func createReq(r *utils.Request) error {
	// здесь доп.обработки не требуется
	return nil
}
