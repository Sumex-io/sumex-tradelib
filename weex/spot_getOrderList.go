package weex

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol    *string
	orderType *entity.OrderType
	limit     *int
}

func (s *spot_getOrderList) Symbol(symbol string) *spot_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *spot_getOrderList) OrderType(orderType entity.OrderType) *spot_getOrderList {
	s.orderType = &orderType
	return s
}

func (s *spot_getOrderList) Limit(limit int) *spot_getOrderList {
	s.limit = &limit
	return s
}

func (s *spot_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_OrdersList, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v3/openOrders",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.limit != nil {
		m["limit"] = *s.limit
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ []spot_orderList
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	res = s.convert.convertOrderList(answ)

	if s.orderType != nil {
		f := make([]entity.Spot_OrdersList, 0, len(res))
		want := strings.ToUpper(string(*s.orderType))
		for _, item := range res {
			if strings.ToUpper(item.Type) == want {
				f = append(f, item)
			}
		}
		res = f
	}

	return res, nil
}

type spot_orderList struct {
	Symbol              string `json:"symbol"`
	OrderId             int64  `json:"orderId"`
	ClientOrderId       string `json:"clientOrderId"`
	Price               string `json:"price"`
	OrigQty             string `json:"origQty"`
	ExecutedQty         string `json:"executedQty"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	Status              string `json:"status"`
	TimeInForce         string `json:"timeInForce"`
	Type                string `json:"type"`
	Side                string `json:"side"`
	Time                int64  `json:"time"`
	UpdateTime          int64  `json:"updateTime"`
	IsWorking           bool   `json:"isWorking"`
}

func (o spot_orderList) OrderIDString() string {
	return strconv.FormatInt(o.OrderId, 10)
}
