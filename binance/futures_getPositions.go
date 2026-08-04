package binance

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============GetPositions=================
type futures_getPositions struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	isCOINM bool
	convert futures_converts
}

func (s *futures_getPositions) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_Positions, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/fapi/v3/positionRisk",
		SecType:  utils.SecTypeSigned,
	}

	if s.isCOINM {
		r.Endpoint = "/dapi/v1/positionRisk"
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := []futures_Position{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	return s.convert.convertPositions(answ), nil
}

type futures_Position struct {
	Symbol           string `json:"symbol"`
	PositionSide     string `json:"positionSide"`
	PositionAmt      string `json:"positionAmt"`
	EntryPrice       string `json:"entryPrice"`
	Leverage         string `json:"leverage"`
	MarkPrice        string `json:"markPrice"`
	UnRealizedProfit string `json:"unRealizedProfit"`
	Notional         string `json:"notional"`
	NotionalValue    string `json:"notionalValue"`
	IsolatedMargin   string `json:"isolatedMargin"`
	UpdateTime       int64  `json:"updateTime"`
}
