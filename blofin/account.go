package blofin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ===================GetAccountInfo==================

type getAccountInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert account_converts
}

func (s *getAccountInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.AccountInformation, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/user/query-apikey",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result accountInfo `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	// на всякий случай проверим, что uid не пустой
	if answ.Result.UID == "" {
		return res, errors.New("Zero Answer")
	}

	return s.convert.convertAccountInfo(answ.Result), nil
}

// структура под /api/v1/user/query-apikey
type accountInfo struct {
	UID          string   `json:"uid"`
	APIName      string   `json:"apiName"`
	APIKey       string   `json:"apiKey"`
	ReadOnly     int      `json:"readOnly"` // 0: read+write, 1: read only
	IPs          []string `json:"ips"`
	Type         int      `json:"type"`       // 1: transaction, 2: connect to third-party
	ExpireTime   string   `json:"expireTime"` // ms timestamp
	CreateTime   string   `json:"createTime"` // ms timestamp
	ReferralCode string   `json:"referralCode"`
	ParentUID    string   `json:"parentUid"`
}

// =================== WebSocket Sign Auth ===================

type signAuthStream struct {
	sec       string
	timeStamp *int64
}

func (s *signAuthStream) TimeStamp(timeStamp int64) *signAuthStream {
	s.timeStamp = &timeStamp
	return s
}

// Do возвращает подпись для WebSocket login
// Для Blofin:
//
//	path = "/users/self/verify"
//	method = "GET"
//	sign = Base64( HMAC-SHA256(prehash).hex() )
//	prehash = path + method + timestamp + nonce
//
// Здесь nonce = timestamp (как в их примере)
func (s *signAuthStream) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.SignAuthStream, err error) {
	sf, err := utils.SignFunc(utils.KeyTypeHmac)
	if err != nil {
		return res, err
	}

	var t int64
	if s.timeStamp != nil {
		t = *s.timeStamp
	} else {
		// если время не передали, возьмём текущее
		t = utils.CurrentTimestamp()
	}

	tsStr := fmt.Sprintf("%d", t)
	nonce := tsStr // Blofin пример делает nonce = timestamp, делаем так же

	const path = "/users/self/verify"
	const method = "GET"

	// prehash: path + method + timestamp + nonce
	raw := path + method + tsStr + nonce

	// HMAC-SHA256 → hex-строка
	hexStr, err := sf(s.sec, raw)
	if err != nil {
		return res, err
	}

	// Base64 от hex-строки
	res.Signature = base64.StdEncoding.EncodeToString([]byte(*hexStr))
	return res, nil
}
