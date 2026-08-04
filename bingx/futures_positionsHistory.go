package bingx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_positionsHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64
}

func (s *futures_positionsHistory) Symbol(symbol string) *futures_positionsHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_positionsHistory) StartTime(startTime int64) *futures_positionsHistory {
	s.startTime = &startTime
	return s
}

func (s *futures_positionsHistory) EndTime(endTime int64) *futures_positionsHistory {
	s.endTime = &endTime
	return s
}

func (s *futures_positionsHistory) Limit(limit int64) *futures_positionsHistory {
	s.limit = &limit
	return s
}

func (s *futures_positionsHistory) Page(page int64) *futures_positionsHistory {
	s.page = &page
	return s
}

func (s *futures_positionsHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_PositionsHistory, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/openApi/swap/v1/trade/positionHistory",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	if s.limit != nil && *s.limit > 0 {
		m["pageSize"] = *s.limit
	}
	if s.page != nil && *s.page > 0 {
		m["pageIndex"] = *s.page
	}
	if s.startTime != nil {
		m["startTs"] = *s.startTime
	}
	if s.endTime != nil {
		m["endTs"] = *s.endTime
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result struct {
			PositionHistory []futures_PositionsHistory_Response `json:"positionHistory"`
		} `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertPositionsHistory(answ.Result.PositionHistory), nil
}

type futures_PositionsHistory_Response struct {
	Symbol             string `json:"symbol"`
	PositionId         string `json:"positionId"`
	Isolated           bool   `json:"isolated"`
	PositionSide       string `json:"positionSide"`
	AvgPrice           string `json:"avgPrice"`
	AvgClosePrice      string `json:"avgClosePrice"`
	RealisedProfit     string `json:"realisedProfit"`
	NetProfit          string `json:"netProfit"`
	PositionAmt        string `json:"positionAmt"`
	ClosePositionAmt   string `json:"closePositionAmt"`
	PositionCommission string `json:"positionCommission"`
	TotalFunding       string `json:"totalFunding"`
	Leverage           int64  `json:"leverage"`

	OpenTime   int64 `json:"openTime"`
	UpdateTime int64 `json:"updateTime"`
}
