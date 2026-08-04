package weex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

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
		Endpoint: "/api/v3/account/",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ accountInfo
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if answ.UID == 0 {
		return res, errors.New("zero answer")
	}

	return s.convert.convertAccountInfo(answ), nil
}

type accountInfo struct {
	MakerCommission            int                    `json:"makerCommission"`
	TakerCommission            int                    `json:"takerCommission"`
	CommissionRates            accountCommissionRates `json:"commissionRates"`
	CanTrade                   bool                   `json:"canTrade"`
	CanWithdraw                bool                   `json:"canWithdraw"`
	CanDeposit                 bool                   `json:"canDeposit"`
	Brokered                   bool                   `json:"brokered"`
	RequireSelfTradePrevention bool                   `json:"requireSelfTradePrevention"`
	PreventSor                 bool                   `json:"preventSor"`
	UpdateTime                 int64                  `json:"updateTime"`
	AccountType                string                 `json:"accountType"`
	Balances                   []accountBalance       `json:"balances"`
	Permissions                []string               `json:"permissions"`
	UID                        int64                  `json:"uid"`
}

type accountCommissionRates struct {
	Maker string `json:"maker"`
	Taker string `json:"taker"`
}

type accountBalance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

// ===============ACCOUNT CONVERT=================

func (c *account_converts) convertAccountInfo(in accountInfo) (out entity.AccountInformation) {
	out.UID = utils.Int64ToString(in.UID)

	// Это spot endpoint, значит спот-доступ у аккаунта есть.
	out.PermSpot = true
	out.PermFutures = true

	out.CanTrade = in.CanTrade

	// Явного "CanRead" у WEEX нет, поэтому считаем чтение доступным,
	// если запрос account info вообще успешно отработал.
	out.CanRead = true

	// Дополнительно учитываем permissions, если биржа их вернула.
	for _, p := range in.Permissions {
		up := strings.ToUpper(p)
		if strings.Contains(up, "SPOT") {
			out.PermSpot = true
		}
		if strings.Contains(up, "TRADING") || strings.Contains(up, "TRADE") {
			out.CanTrade = out.CanTrade || in.CanTrade
		}
	}

	return out
}

type signAuthStream struct {
	sec       string
	timeStamp *int64
}

func (s *signAuthStream) TimeStamp(timeStamp int64) *signAuthStream {
	s.timeStamp = &timeStamp
	return s
}

func (s *signAuthStream) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.SignAuthStream, err error) {
	sf, err := utils.SignFunc(utils.KeyTypeHmacBase64)
	if err != nil {
		return res, err
	}

	t := int64(0)
	if s.timeStamp != nil {
		t = *s.timeStamp
	}

	raw := fmt.Sprintf("%d%s", t, "/v3/ws/private")
	sign, err := sf(s.sec, raw)
	if err != nil {
		return res, err
	}

	res.Signature = *sign
	return res, nil
}
