package bybit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getPositions struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol   *string
	category *string
}

func (s *futures_getPositions) Symbol(symbol string) *futures_getPositions {
	s.symbol = &symbol
	return s
}

func (s *futures_getPositions) Category(category string) *futures_getPositions {
	s.category = &category
	return s
}

func (s *futures_getPositions) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_Positions, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v5/position/list",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"category": "linear", "limit": 200}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	} else {
		m["settleCoin"] = "USDT"
	}
	if s.category != nil {
		m["category"] = *s.category
		if *s.category == "inverse" {
			delete(m, "settleCoin")
		}
	}
	// settleCoin need check
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result struct {
			List []futures_Position `json:"list"`
		} `json:"result"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	res = s.convert.convertPositions(answ.Result.List)
	return res, nil
}

type futures_Position struct {
	Symbol         string `json:"symbol"`
	Side           string `json:"side"`
	PositionIdx    int64  `json:"positionIdx"`
	Size           string `json:"size"`
	Leverage       string `json:"leverage"`
	AvgPrice       string `json:"avgPrice"`
	MarkPrice      string `json:"markPrice"`
	UnrealisedPnl  string `json:"unrealisedPnl"`
	CurRealisedPnl string `json:"curRealisedPnl"`
	PositionValue  string `json:"positionValue"`

	CreatedTime string `json:"createdTime"`
	UpdatedTime string `json:"updatedTime"`
}
