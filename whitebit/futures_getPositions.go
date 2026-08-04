package whitebit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ============== GetPositions (WhiteBIT Futures) =============

type futures_getPositions struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol *string // market, например "BTC_PERP" или "BTC_USDT"
}

func (s *futures_getPositions) Symbol(symbol string) *futures_getPositions {
	s.symbol = &symbol
	return s
}

func (s *futures_getPositions) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_Positions, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/collateral-account/positions/open",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}
	if s.symbol != nil && *s.symbol != "" {
		// В доке параметр называется market
		m["market"] = *s.symbol
	}

	// WhiteBIT всё подписывает через body (nonce + request + payload),
	// поэтому используем форму (SetFormParams), а не query.
	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// Ответ — простой массив
	var answ []futures_positionWB
	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertPositions(answ), nil
}

// Структура под /api/v4/collateral-account/positions/open
type futures_positionWB struct {
	PositionId        int64   `json:"positionId"`
	Market            string  `json:"market"`
	OpenDate          float64 `json:"openDate"`
	ModifyDate        float64 `json:"modifyDate"`
	Amount            string  `json:"amount"`
	BasePrice         string  `json:"basePrice"`
	LiquidationPrice  *string `json:"liquidationPrice"`
	LiquidationState  *string `json:"liquidationState"`
	Pnl               string  `json:"pnl"`        // нереализованный PnL в money
	PnlPercent        string  `json:"pnlPercent"` // %
	Margin            string  `json:"margin"`     // margin в money
	FreeMargin        string  `json:"freeMargin"` // свободная маржа
	Funding           string  `json:"funding"`    // уже начисленный funding
	UnrealizedFunding string  `json:"unrealizedFunding"`
	PositionSide      string  `json:"positionSide"` // LONG / SHORT / BOTH
	// tpsl можно добавить при необходимости, пока не используется:
	// Tpsl *struct {
	// 	TakeProfitId int64  `json:"takeProfitId"`
	// 	TakeProfit   string `json:"takeProfit"`
	// 	StopLossId   int64  `json:"stopLossId"`
	// 	StopLoss     string `json:"stopLoss"`
	// } `json:"tpsl"`
}
