package okx

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

	orderID *string
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

func (s *futures_positionsHistory) OrderID(orderID string) *futures_positionsHistory {
	s.orderID = &orderID
	return s
}

func (s *futures_positionsHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_PositionsHistory, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v5/account/positions-history",
		SecType:  utils.SecTypeSigned,
	}

	// m := utils.Params{"instType": "SWAP", "type": "3"}
	m := utils.Params{"instType": "SWAP"}
	if s.symbol != nil {
		m["instId"] = *s.symbol
	}
	if s.limit != nil && *s.limit > 0 {
		m["limit"] = *s.limit
	}
	// if s.page != nil && *s.page > 0 {
	// 	m["pageIndex"] = *s.page
	// }
	if s.startTime != nil {
		// m["after"] = *s.startTime
		m["before"] = *s.startTime
	}
	if s.endTime != nil {
		// m["before"] = *s.endTime
		m["after"] = *s.endTime
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []futures_PositionsHistory_Response `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertPositionsHistory(answ.Result), nil
}

type futures_PositionsHistory_Response struct {
	InstId        string `json:"instId"`
	PosId         string `json:"posId"`
	MgnMode       string `json:"mgnMode"`
	PosSide       string `json:"posSide"`
	Type          string `json:"type"`
	Direction     string `json:"direction"`
	OpenAvgPx     string `json:"openAvgPx"`
	CloseAvgPx    string `json:"closeAvgPx"`
	OpenMaxPos    string `json:"openMaxPos"`
	Lever         string `json:"lever"`
	CloseTotalPos string `json:"closeTotalPos"`
	Pnl           string `json:"pnl"`
	RealizedPnl   string `json:"realizedPnl"`
	Fee           string `json:"fee"`
	FundingFee    string `json:"fundingFee"`
	CTime         string `json:"cTime"`
	UTime         string `json:"uTime"`
}
