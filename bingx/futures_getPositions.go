package bingx

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

	symbol *string
}

func (s *futures_getPositions) Symbol(symbol string) *futures_getPositions {
	s.symbol = &symbol
	return s
}

func (s *futures_getPositions) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_Positions, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/openApi/swap/v2/user/positions",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []futures_Position `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	res = s.convert.convertPositions(answ.Result)
	return res, nil
}

type futures_Position struct {
	Symbol           string `json:"symbol"`
	PositionId       string `json:"positionId"`
	PositionAmt      string `json:"positionAmt"`
	AvailableAmt     string `json:"availableAmt"`
	PositionSide     string `json:"positionSide"`
	Isolated         bool   `json:"isolated"`
	AvgPrice         string `json:"avgPrice"`
	Leverage         int64  `json:"leverage"`
	UnrealizedProfit string `json:"unrealizedProfit"`
	RealisedProfit   string `json:"realisedProfit"`
	MarkPrice        string `json:"markPrice"`
	PositionValue    string `json:"positionValue"`
	OnlyOnePosition  bool   `json:"onlyOnePosition"`

	CreateTime int64 `json:"createTime"`
	UpdateTime int64 `json:"updateTime"`
}
