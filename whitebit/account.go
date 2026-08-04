package whitebit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// =================== GetAccountInfo (futures / collateral account) ===================

type getAccountInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert account_converts
}

func (s *getAccountInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.AccountInformation, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/collateral-account/summary",
		SecType:  utils.SecTypeSigned,
	}

	// По доке: достаточно request + nonce, форму можем оставить пустой,
	// whitebit/request.go сам добавит "request" и "nonce" в payload.
	r.SetFormParams(utils.Params{})

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ whitebitAccountSummary
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	// Если equity пустая строка — что-то странно, подстрахуемся.
	if answ.Equity == "" {
		return res, errors.New("empty account summary response")
	}

	return s.convert.convertAccountInfo(answ), nil
}

// минимальная структура под /api/v4/collateral-account/summary
type whitebitAccountSummary struct {
	Equity                       string `json:"equity"`
	Margin                       string `json:"margin"`
	FreeMargin                   string `json:"freeMargin"`
	UnrealizedFunding            string `json:"unrealizedFunding"`
	PnL                          string `json:"pnl"`
	Leverage                     int    `json:"leverage"`
	MarginFraction               string `json:"marginFraction"`
	MaintenanceMarginRequirement string `json:"maintenanceMarginRequirement"`
}
