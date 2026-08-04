package gateio

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

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
	settle    *string
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
		Endpoint: "/api/v4/futures/{settle}/position_close",
		SecType:  utils.SecTypeSigned,
	}

	settleDefault := "usdt"

	if s.settle == nil {
		s.settle = &settleDefault
	}

	r.Endpoint = strings.Replace(r.Endpoint, "{settle}", *s.settle, 1)

	multOff := int64(0)
	m := utils.Params{}
	if s.symbol != nil {
		m["contract"] = *s.symbol
	}
	if s.limit != nil && *s.limit > 0 {
		m["limit"] = *s.limit
		multOff = *s.limit
	}
	if s.page != nil && *s.page > 0 {
		m["offset"] = (*s.page - 1) * multOff
	}
	if s.startTime != nil {
		m["from"] = *s.startTime
	}
	if s.endTime != nil {
		m["to"] = *s.endTime
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := []futures_PositionsHistory_Response{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertPositionsHistory(answ), nil
}

type futures_PositionsHistory_Response struct {
	Contract string `json:"contract"`
	// PositionId         string `json:"positionId"`
	// Isolated           bool   `json:"isolated"`
	Side        string `json:"side"`
	Long_price  string `json:"long_price"`
	Short_price string `json:"short_price"`
	Pnl         string `json:"pnl"`
	Pnl_pnl     string `json:"pnl_pnl"`
	Max_size    string `json:"max_size"`
	Accum_size  string `json:"accum_size"`
	Pnl_fee     string `json:"pnl_fee"`
	Pnl_fund    string `json:"pnl_fund"`
	Leverage    string `json:"leverage"`

	First_open_time int64 `json:"first_open_time"`
	Time            int64 `json:"time"`
}
