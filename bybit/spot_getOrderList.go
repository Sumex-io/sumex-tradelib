package bybit

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
}

func (s *spot_getOrderList) Symbol(symbol string) *spot_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *spot_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_OrdersList, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v5/order/realtime",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"category": "spot",
		"limit":    50,
	}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result struct {
			List []spot_orderList `json:"list"`
		} `json:"result"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrderList(answ.Result.List), nil
}

type spot_orderList struct {
	Symbol      string `json:"symbol"`
	OrderType   string `json:"orderType"`
	OrderLinkId string `json:"orderLinkId"`
	OrderId     string `json:"orderId"`
	OrderStatus string `json:"orderStatus"`
	Price       string `json:"price"`
	CreatedTime string `json:"createdTime"`
	UpdatedTime string `json:"updatedTime"`
	Side        string `json:"side"`
	Qty         string `json:"qty"`
	CumExecQty  string `json:"cumExecQty"`
}
