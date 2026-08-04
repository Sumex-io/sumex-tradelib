package blofin

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

	state  *string
	after  *string
	before *string
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

// Page оставляем для совместимости с общим интерфейсом.
// У Blofin в этом endpoint нет page/pageIndex, там cursor-пагинация через after/before.
func (s *futures_positionsHistory) Page(page int64) *futures_positionsHistory {
	s.page = &page
	return s
}

// В общем интерфейсе это OrderID, но для Blofin positions-history это positionId.
func (s *futures_positionsHistory) OrderID(orderID string) *futures_positionsHistory {
	s.orderID = &orderID
	return s
}

func (s *futures_positionsHistory) State(state string) *futures_positionsHistory {
	s.state = &state
	return s
}

func (s *futures_positionsHistory) After(after string) *futures_positionsHistory {
	s.after = &after
	return s
}

func (s *futures_positionsHistory) Before(before string) *futures_positionsHistory {
	s.before = &before
	return s
}

func (s *futures_positionsHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_PositionsHistory, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/account/positions-history",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.orderID != nil {
		m["positionId"] = *s.orderID
	}
	if s.symbol != nil {
		m["instId"] = *s.symbol
	}
	if s.state != nil {
		m["state"] = *s.state
	}
	if s.after != nil {
		m["after"] = *s.after
	}
	if s.before != nil {
		m["before"] = *s.before
	}
	if s.startTime != nil {
		m["begin"] = *s.startTime
	}
	if s.endTime != nil {
		m["end"] = *s.endTime
	}
	if s.limit != nil && *s.limit > 0 {
		m["limit"] = *s.limit
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []futures_PositionsHistory_Response `json:"data"`
	}

	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertPositionsHistory(answ.Result), nil
}

type futures_PositionsHistory_Response struct {
	HistoryId            string `json:"historyId"`
	PositionId           string `json:"positionId"`
	InstId               string `json:"instId"`
	InstType             string `json:"instType"`
	Side                 string `json:"side"`
	MarginMode           string `json:"marginMode"`
	PositionSide         string `json:"positionSide"`
	ClosePositions       string `json:"closePositions"`
	MaxPositions         string `json:"maxPositions"`
	LiquidationPositions string `json:"liquidationPositions"`
	OpenAveragePrice     string `json:"openAveragePrice"`
	CloseAveragePrice    string `json:"closeAveragePrice"`
	CreateTime           string `json:"createTime"`
	UpdateTime           string `json:"updateTime"`
	Leverage             string `json:"leverage"`
	RealizedPnl          string `json:"realizedPnl"`
	RealizedPnlRatio     string `json:"realizedPnlRatio"`
	Fee                  string `json:"fee"`
}
