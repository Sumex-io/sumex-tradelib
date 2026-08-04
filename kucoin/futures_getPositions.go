package kucoin

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
	convert futures_converts
}

func (s *futures_getPositions) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_Positions, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/positions",
		SecType:  utils.SecTypeSigned,
	}

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
	return s.convert.convertPositions(answ.Result), nil
}

type futures_Position struct {
	ID               string  `json:"id"`
	Symbol           string  `json:"symbol"`
	MarginMode       string  `json:"marginMode"`
	PositionSide     string  `json:"positionSide"`
	CrossMode        bool    `json:"crossMode"`
	IsOpen           bool    `json:"isOpen"`
	MarkPrice        float64 `json:"markPrice"`
	CurrentQty       float64 `json:"currentQty"`
	CurrentCost      float64 `json:"currentCost"`
	RealisedPnl      float64 `json:"realisedPnl"`
	UnrealisedPnl    float64 `json:"unrealisedPnl"`
	AvgEntryPrice    float64 `json:"avgEntryPrice"`
	Leverage         int64   `json:"leverage"`
	OpeningTimestamp int64   `json:"openingTimestamp"`
	CurrentTimestamp int64   `json:"currentTimestamp"`
}
