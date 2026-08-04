package okx

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============getOrderList=================
type futures_getOrderList struct {
	convert futures_converts

	callAPI   func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	symbol    *string
	instType  *string
	orderType *entity.OrderType
	limit     *int
}

func (s *futures_getOrderList) Symbol(symbol string) *futures_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *futures_getOrderList) InstType(instType string) *futures_getOrderList {
	s.instType = &instType
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
	// 1) обычные pending-ордера
	{
		r := &utils.Request{
			Method:   http.MethodGet,
			Endpoint: "/api/v5/trade/orders-pending",
			SecType:  utils.SecTypeSigned,
		}

		m := utils.Params{
			"instType": "SWAP",
		}

		if s.symbol != nil {
			m["instId"] = *s.symbol
		}
		if s.instType != nil {
			m["instType"] = *s.instType
		}
		if s.limit != nil {
			m["limit"] = *s.limit
		}
		if s.orderType != nil {
			m["ordType"] = strings.ToLower(string(*s.orderType))
		}

		r.SetParams(m)

		data, _, e := s.callAPI(ctx, r, opts...)
		if e != nil {
			return res, e
		}

		var answ struct {
			Result []futures_orderList `json:"data"`
		}

		if e := json.Unmarshal(data, &answ); e != nil {
			return res, e
		}

		res = append(res, s.convert.convertOrderList(answ.Result)...)
	}

	// 2) algo pending (TP/SL conditional), которые создавались через /trade/order-algo
	// Важно: эти ордера НЕ попадают в orders-pending.
	{
		r := &utils.Request{
			Method:   http.MethodGet,
			Endpoint: "/api/v5/trade/orders-algo-pending",
			SecType:  utils.SecTypeSigned,
		}

		m := utils.Params{
			"instType": "SWAP",
			"ordType":  "conditional", // TP/SL ветка в твоём placeOrder выставляет именно conditional
		}

		if s.symbol != nil {
			m["instId"] = *s.symbol
		}
		if s.instType != nil {
			m["instType"] = *s.instType
		}
		if s.limit != nil {
			m["limit"] = *s.limit
		}

		r.SetParams(m)

		data, _, e := s.callAPI(ctx, r, opts...)
		if e != nil {
			return res, e
		}

		var answ struct {
			Result []futures_algoOrder `json:"data"`
		}

		if e := json.Unmarshal(data, &answ); e != nil {
			return res, e
		}

		res = append(res, s.convert.convertAlgoOrderList(answ.Result)...)
	}

	return res, nil
}

// ===== обычные pending =====

type futures_orderList struct {
	InstId         string                             `json:"instId"`
	OrdId          string                             `json:"ordId"`
	ClOrdId        string                             `json:"clOrdId"`
	Px             string                             `json:"px"`
	Sz             string                             `json:"sz"`
	AttachAlgoOrds []futures_orderList_attachAlgoOrds `json:"attachAlgoOrds"` // ВАЖНО: lowercase
	PosSide        string                             `json:"posSide"`
	OrdType        string                             `json:"ordType"`
	TdMode         string                             `json:"tdMode"`
	InstType       string                             `json:"instType"`
	Lever          string                             `json:"lever"`
	Side           string                             `json:"side"`
	State          string                             `json:"state"`
	FillSz         string                             `json:"fillSz"`
	IsTpLimit      string                             `json:"isTpLimit"`
	UTime          string                             `json:"uTime"`
	CTime          string                             `json:"cTime"`
}

type futures_orderList_attachAlgoOrds struct {
	AttachAlgoId string `json:"attachAlgoId"`
	SlOrdPx      string `json:"slOrdPx"`
	SlTriggerPx  string `json:"slTriggerPx"`
	TpOrdPx      string `json:"tpOrdPx"`
	TpTriggerPx  string `json:"tpTriggerPx"`
}

// ===== algo pending (conditional TP/SL) =====
// Поля у OKX могут отличаться в зависимости от типа algo, но для conditional
// нам обычно хватает этих.
type futures_algoOrder struct {
	AlgoId      string `json:"algoId"`
	AlgoClOrdId string `json:"algoClOrdId"`
	InstId      string `json:"instId"`
	InstType    string `json:"instType"`
	OrdType     string `json:"ordType"` // conditional
	Side        string `json:"side"`
	PosSide     string `json:"posSide"`
	Sz          string `json:"sz"`
	State       string `json:"state"`
	CTime       string `json:"cTime"`
	UTime       string `json:"uTime"`

	// TP/SL trigger fields (у conditional они приходят так)
	TpTriggerPx string `json:"tpTriggerPx"`
	SlTriggerPx string `json:"slTriggerPx"`

	// иногда встречаются альтернативные поля
	TriggerPx string `json:"triggerPx"`
}
