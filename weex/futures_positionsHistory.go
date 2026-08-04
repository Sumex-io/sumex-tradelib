package weex

import (
	"context"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_positionsHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64
	orderID   *string
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
	return res, errors.New("weex futures positions history is not supported as a dedicated endpoint")
}
