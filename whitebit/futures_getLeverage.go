package whitebit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// =================== GetLeverage (account-level) ===================

type futures_getLeverage struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts
}

func (s *futures_getLeverage) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_Leverage, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/collateral-account/summary",
		SecType:  utils.SecTypeSigned,
	}

	// По доке: только request + nonce, без доп. параметров.
	// Наш whitebit request.go сам подставит request/nonce,
	// мы просто передаём пустую форму.
	r.SetFormParams(utils.Params{})

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ futures_leverageSummary
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertLeverage(answ), nil
}

// структура под /api/v4/collateral-account/summary
type futures_leverageSummary struct {
	Equity                       string `json:"equity"`
	Margin                       string `json:"margin"`
	FreeMargin                   string `json:"freeMargin"`
	UnrealizedFunding            string `json:"unrealizedFunding"`
	PnL                          string `json:"pnl"`
	Leverage                     int    `json:"leverage"`
	MarginFraction               string `json:"marginFraction"`
	MaintenanceMarginRequirement string `json:"maintenanceMarginRequirement"`
}
