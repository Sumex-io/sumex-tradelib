package bybit

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
	category  *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64
	cursor    *string

	orderID *string
}

func (s *futures_positionsHistory) Symbol(symbol string) *futures_positionsHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_positionsHistory) Category(category string) *futures_positionsHistory {
	s.category = &category
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

func (s *futures_positionsHistory) Cursor(cursor string) *futures_positionsHistory {
	s.cursor = &cursor
	return s
}

func (s *futures_positionsHistory) OrderID(orderID string) *futures_positionsHistory {
	s.orderID = &orderID
	return s
}

func (s *futures_positionsHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_PositionsHistory, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v5/position/closed-pnl",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"category": "linear", "limit": 100}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	if s.category != nil {
		m["category"] = *s.category
	}
	if s.limit != nil && *s.limit > 0 {
		m["limit"] = *s.limit
	}
	if s.cursor != nil && *s.cursor != "" {
		m["cursor"] = *s.cursor
	}
	if s.startTime != nil {
		m["startTime"] = *s.startTime
	}
	if s.endTime != nil {
		m["endTime"] = *s.endTime
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result struct {
			List []futures_PositionsHistory_Response `json:"list"`
		} `json:"result"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertPositionsHistory(answ.Result.List), nil
}

type futures_PositionsHistory_Response struct {
	Symbol        string `json:"symbol"`
	Side          string `json:"side"`
	OrderId       string `json:"orderId"`
	PositionIdx   int64  `json:"positionIdx"`
	Qty           string `json:"qty"`
	ClosedSize    string `json:"closedSize"`
	AvgEntryPrice string `json:"avgEntryPrice"`
	AvgExitPrice  string `json:"avgExitPrice"`
	ClosedPnl     string `json:"closedPnl"`
	OpenFee       string `json:"openFee"`
	CloseFee      string `json:"closeFee"`
	Leverage      string `json:"leverage"`

	CreatedTime string `json:"createdTime"`
	UpdatedTime string `json:"updatedTime"`
}
