package whitebit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============GetOrderList (conditional orders)=================
type futures_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol *string
	limit  *int
	offset *int
}

func (s *futures_getOrderList) Symbol(symbol string) *futures_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *futures_getOrderList) Limit(limit int) *futures_getOrderList {
	s.limit = &limit
	return s
}

func (s *futures_getOrderList) Offset(offset int) *futures_getOrderList {
	s.offset = &offset
	return s
}

func (s *futures_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersList, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/orders",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	// market необязателен — если есть, фильтруем по одному символу
	if s.symbol != nil {
		// Для фьючей WhiteBit формат и так "BTC_PERP", так что без конвертации
		m["market"] = *s.symbol
	}
	if s.limit != nil {
		m["limit"] = *s.limit
	}
	if s.offset != nil {
		m["offset"] = *s.offset
	}

	if len(m) > 0 {
		// WhiteBit всё несёт в body
		r.SetFormParams(m)
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// /api/v4/orders возвращает просто массив ордеров
	var answ []futures_orderListWB

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrderList(answ), nil
}

type futures_orderListWB struct {
	OrderId       int64   `json:"orderId"`
	ClientOrderId string  `json:"clientOrderId"`
	Market        string  `json:"market"`
	Side          string  `json:"side"`
	Type          string  `json:"type"`
	Timestamp     float64 `json:"timestamp"`
	DealMoney     string  `json:"dealMoney"`
	DealStock     string  `json:"dealStock"`
	Amount        string  `json:"amount"`
	Left          string  `json:"left"`
	DealFee       string  `json:"dealFee"`
	IOC           bool    `json:"ioc"`
	Status        string  `json:"status"`
	Stp           string  `json:"stp"`
	PositionSide  string  `json:"positionSide"`
	RPI           bool    `json:"rpi"`
	Price         string  `json:"price"`
	PostOnly      bool    `json:"postOnly"`
}
