package bitget

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
		Endpoint: "/api/v2/spot/trade/unfilled-orders",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []spot_orderList `json:"data"`
	}
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrderList(answ.Result), nil
}

type spot_orderList struct {
	Symbol      string `json:"symbol"`
	OrderId     string `json:"orderId"`
	ClientOid   string `json:"clientOid"`
	Create_time int64  `json:"create_time_ms"`
	Update_time int64  `json:"update_time_ms"`
	Status      string `json:"status"`
	OrderType   string `json:"orderType"`
	Side        string `json:"side"`
	Size        string `json:"size"`
	BaseVolume  string `json:"baseVolume"`
	PriceAvg    string `json:"priceAvg"`
	CTime       string `json:"cTime"`
	UTime       string `json:"uTime"`
}
