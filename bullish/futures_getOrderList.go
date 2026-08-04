package bullish

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	orderType *entity.OrderType
	limit     *int
	uid       *string
}

func (s *futures_getOrderList) Symbol(symbol string) *futures_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *futures_getOrderList) UID(uid string) *futures_getOrderList {
	s.uid = &uid
	return s
}

func (s *futures_getOrderList) OrderType(orderType entity.OrderType) *futures_getOrderList {
	s.orderType = &orderType
	return s
}

func (s *futures_getOrderList) Limit(limit int) *futures_getOrderList {
	s.limit = &limit
	return s
}

func (s *futures_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersList, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/trading-api/v2/orders",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"status": "OPEN"}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.uid != nil {
		m["tradingAccountId"] = *s.uid
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	answ := []futures_orderList{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	res = s.convert.convertOrderList(answ)
	return res, nil
}

type futures_orderList struct {
	Symbol        string `json:"symbol"`
	OrderId       string `json:"orderId"`
	ClientOrderId string `json:"clientOrderId"`
	// PositionID    int64  `json:"positionID"`
	Side string `json:"side"`
	// PositionSide  string `json:"positionSide"`
	Type             string `json:"type"`
	Quantity         string `json:"quantity"`
	QuantityFilled   string `json:"quantityFilled"`
	Price            string `json:"price"`
	AverageFillPrice string `json:"averageFillPrice"`
	// Leverage      string `json:"leverage"`
	Status             string `json:"status"`
	CreatedAtTimestamp string `json:"createdAtTimestamp"`
	// UpdateTime    int64  `json:"updateTime"`
}
