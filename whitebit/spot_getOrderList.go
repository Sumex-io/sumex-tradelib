package whitebit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol *string
	limit  *int
	offset *int
}

func (s *spot_getOrderList) Symbol(symbol string) *spot_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *spot_getOrderList) Limit(limit int) *spot_getOrderList {
	s.limit = &limit
	return s
}

func (s *spot_getOrderList) Offset(offset int) *spot_getOrderList {
	s.offset = &offset
	return s
}

func (s *spot_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_OrdersList, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/orders",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	// фильтр по рынку, если передали
	if s.symbol != nil {
		m["market"] = *s.symbol
	}

	// limit / offset не обязательны
	if s.limit != nil && *s.limit > 0 {
		m["limit"] = *s.limit
	}
	if s.offset != nil && *s.offset >= 0 {
		m["offset"] = *s.offset
	}

	// WhiteBIT ждёт JSON body
	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ []spot_orderListWB
	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertSpotOrderList(answ), nil
}

// структура под /api/v4/orders для СПОТА
// (для фьючей мы использовали похожую, но с positionSide и "margin limit")
type spot_orderListWB struct {
	OrderID       int64   `json:"orderId"`
	ClientOrderID string  `json:"clientOrderId"`
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
	Price         string  `json:"price,omitempty"`
	PostOnly      bool    `json:"postOnly"`
	RPI           bool    `json:"rpi,omitempty"`
	STP           string  `json:"stp,omitempty"`
}
