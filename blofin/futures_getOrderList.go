package blofin

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============GetOrderList (active orders)=================
type futures_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	orderType *entity.OrderType
	state     *string
	// при необходимости можно будет добавить after/before/limit
}

func (s *futures_getOrderList) Symbol(symbol string) *futures_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *futures_getOrderList) OrderType(orderType entity.OrderType) *futures_getOrderList {
	s.orderType = &orderType
	return s
}

// state: "live", "partially_filled" и т.п. (как в доках Blofin)
func (s *futures_getOrderList) State(state string) *futures_getOrderList {
	s.state = &state
	return s
}

func (s *futures_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersList, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/trade/orders-pending",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["instId"] = *s.symbol
	}

	if s.orderType != nil {
		m["orderType"] = strings.ToLower(string(*s.orderType))
	}

	if s.state != nil {
		m["state"] = strings.ToLower(*s.state)
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []futures_orderList `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrderList(answ.Result), nil
}

// структура под Blofin /api/v1/trade/orders-pending
// поля названы по аналогии с place order / history / positions
type futures_orderList struct {
	InstId        string `json:"instId"`
	OrderId       string `json:"orderId"`
	ClientOrderId string `json:"clientOrderId"`
	Price         string `json:"price"`
	Size          string `json:"size"`
	FilledSize    string `json:"filledSize"`
	Side          string `json:"side"`         // buy / sell
	PositionSide  string `json:"positionSide"` // long / short / net
	OrderType     string `json:"orderType"`    // limit / market / ...
	MarginMode    string `json:"marginMode"`   // cross / isolated
	Leverage      string `json:"leverage"`
	State         string `json:"state"`      // live / partially_filled / ...
	CreateTime    string `json:"createTime"` // ms
	UpdateTime    string `json:"updateTime"` // ms
	PositionId    string `json:"positionId"` // если биржа вернёт, заберём; если нет — просто пусто
}
