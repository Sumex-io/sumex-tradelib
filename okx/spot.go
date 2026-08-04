package okx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============TradeCancelOrders=================

// ==============multiCancelOrders=================

type multiCancelOrders struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)

	symbol   *string
	orderIDs *[]string
}

func (s *multiCancelOrders) Symbol(symbol string) *multiCancelOrders {
	s.symbol = &symbol
	return s
}

func (s *multiCancelOrders) OrderIDs(orderIDs []string) *multiCancelOrders {
	s.orderIDs = &orderIDs
	return s
}

func (s *multiCancelOrders) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v5/trade/cancel-batch-orders",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"is_batch": "true",
	}

	if s.symbol != nil && s.orderIDs != nil {
		orderIDs := []ordersIDs{}
		for _, item := range *s.orderIDs {
			orderIDs = append(orderIDs, ordersIDs{
				InstId: *s.symbol,
				OrdId:  item,
			})
		}
		j, err := json.Marshal(orderIDs)
		if err != nil {
			return res, err
		}
		m["is_batch"] = string(j)
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []placeOrder_Response `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ.Result) == 0 {
		return res, errors.New("Zero Answer")
	}

	if answ.Result[0].SCode != "0" {
		return res, errors.New(answ.Result[0].SMsg)
	}
	return convertPlaceOrder(answ.Result), nil
}

type ordersIDs struct {
	InstId string `json:"instId"`
	OrdId  string `json:"ordId"`
}

// // ==============getOrderList=================
// type getOrderList struct {
// 	callAPI   func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
// 	symbol    *string
// 	orderType *entity.OrderType
// 	limit     *int
// }

// func (s *getOrderList) Symbol(symbol string) *getOrderList {
// 	s.symbol = &symbol
// 	return s
// }

// func (s *getOrderList) OrderType(orderType entity.OrderType) *getOrderList {
// 	s.orderType = &orderType
// 	return s
// }

// func (s *getOrderList) Limit(limit int) *getOrderList {
// 	s.limit = &limit
// 	return s
// }

// func (s *getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.OrdersPendingList, err error) {
// 	r := &utils.Request{
// 		Method:   http.MethodGet,
// 		Endpoint: "/api/v5/trade/orders-pending",
// 		SecType:  utils.SecTypeSigned,
// 	}

// 	m := utils.Params{
// 		"instType": "SPOT",
// 	}

// 	if s.symbol != nil {
// 		m["instId"] = *s.symbol
// 	}

// 	if s.limit != nil {
// 		m["limit"] = *s.limit
// 	}

// 	if s.orderType != nil {
// 		m["ordType"] = strings.ToLower(string(*s.orderType))
// 	}

// 	r.SetParams(m)

// 	data, _, err := s.callAPI(ctx, r, opts...)
// 	if err != nil {
// 		return res, err
// 	}
// 	var answ struct {
// 		Result []orderList `json:"data"`
// 	}

// 	err = json.Unmarshal(data, &answ)
// 	if err != nil {
// 		return res, err
// 	}

// 	return convertOrderList(answ.Result), nil
// }

// type orderList struct {
// 	InstId         string                     `json:"instId"`
// 	OrdId          string                     `json:"ordId"`
// 	ClOrdId        string                     `json:"clOrdId"`
// 	Px             string                     `json:"px"`
// 	Sz             string                     `json:"sz"`
// 	AttachAlgoOrds []orderList_attachAlgoOrds `json:"AttachAlgoOrds"`
// 	PosSide        string                     `json:"posSide"`
// 	OrdType        string                     `json:"ordType"`
// 	TdMode         string                     `json:"tdMode"`
// 	InstType       string                     `json:"instType"`
// 	Lever          string                     `json:"lever"`
// 	Side           string                     `json:"side"`
// 	State          string                     `json:"state"`
// 	IsTpLimit      string                     `json:"isTpLimit"`
// 	UTime          string                     `json:"uTime"`
// 	CTime          string                     `json:"cTime"`
// }

// type orderList_attachAlgoOrds struct {
// 	AttachAlgoId string `json:"attachAlgoId"`
// 	SlOrdPx      string `json:"slOrdPx"`
// 	SlTriggerPx  string `json:"slTriggerPx"`
// 	TpOrdPx      string `json:"tpOrdPx"`
// 	TpTriggerPx  string `json:"tpTriggerPx"`
// }
