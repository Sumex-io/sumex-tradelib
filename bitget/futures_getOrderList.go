package bitget

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	orderType *entity.OrderType
	limit     *int
}

func (s *futures_getOrderList) Symbol(symbol string) *futures_getOrderList {
	s.symbol = &symbol
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
	// -------------------------
	// 1) обычные открытые ордера
	// -------------------------
	r1 := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v2/mix/order/orders-pending",
		SecType:  utils.SecTypeSigned,
	}

	m1 := utils.Params{"productType": "USDT-FUTURES"}
	if s.symbol != nil {
		m1["symbol"] = *s.symbol
	}
	if s.limit != nil {
		m1["limit"] = *s.limit
	}

	// orderType у Bitget в этом эндпоинте параметром может не поддерживаться (зависит от версии),
	// поэтому фильтрацию делаем после конвертации при необходимости.
	r1.SetParams(m1)

	data1, _, err := s.callAPI(ctx, r1, opts...)
	if err != nil {
		return res, err
	}

	var answ1 struct {
		Result futures_orderList `json:"data"`
	}
	if err := json.Unmarshal(data1, &answ1); err != nil {
		return res, err
	}

	out := s.convert.convertOrderList(answ1.Result)

	// -----------------------------------------
	// 2) TP/SL (plan/trigger) открытые ордера
	// -----------------------------------------
	// Док: Get Pending Trigger Order (plan orders / TP/SL) :contentReference[oaicite:1]{index=1}
	r2 := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v2/mix/order/orders-plan-pending",
		SecType:  utils.SecTypeSigned,
	}

	m2 := utils.Params{
		"productType": "USDT-FUTURES",
		"marginCoin":  "USDT",
		"planType":    "profit_loss",
	}
	if s.symbol != nil {
		m2["symbol"] = *s.symbol
	}
	if s.limit != nil {
		m2["limit"] = *s.limit
	}

	r2.SetParams(m2)

	data2, _, err := s.callAPI(ctx, r2, opts...)
	if err != nil {
		return res, err
	}

	var answ2 struct {
		Result futures_planOrderList `json:"data"`
	}
	if err := json.Unmarshal(data2, &answ2); err != nil {
		return res, err
	}

	out = append(out, s.convert.convertPlanOrderList(answ2.Result)...)

	// -----------------------------------------
	// 3) optional local фильтр по orderType (если нужно)
	// -----------------------------------------
	if s.orderType != nil {
		need := string(*s.orderType)
		filtered := make([]entity.Futures_OrdersList, 0, len(out))
		for _, it := range out {
			// it.Type у вас уже в UPPERCASE
			if it.Type == strings.ToUpper(need) {
				filtered = append(filtered, it)
			}
		}
		out = filtered
	}

	return out, nil
}

type futures_orderList struct {
	Orders []struct {
		Symbol        string `json:"symbol"`
		OrderId       string `json:"orderId"`
		ClientOrderId string `json:"clientOid"`
		Side          string `json:"side"`
		PositionSide  string `json:"posSide"`
		Type          string `json:"orderType"`
		Size          string `json:"size"`
		BaseVolume    string `json:"baseVolume"`
		Price         string `json:"price"`
		AvgPrice      string `json:"priceAvg"`
		Leverage      string `json:"leverage"`
		MarginMode    string `json:"marginMode"`
		TradeSide     string `json:"tradeSide"`
		PosMode       string `json:"posMode"`
		Status        string `json:"status"`
		Time          string `json:"cTime"`
		UpdateTime    string `json:"uTime"`
	} `json:"entrustedList"`
}

// plan/trigger orders (TP/SL и прочие) — структура по docs Bitget Trigger Order -> Get Pending Trigger Order :contentReference[oaicite:2]{index=2}
type futures_planOrderList struct {
	Orders []struct {
		Symbol    string `json:"symbol"`
		OrderId   string `json:"orderId"`
		ClientOid string `json:"clientOid"`

		Side      string `json:"side"`
		PosSide   string `json:"posSide"`
		TradeSide string `json:"tradeSide"`

		PlanType     string `json:"planType"`  // profit_plan / loss_plan / pos_profit / pos_loss / normal_plan ...
		OrderType    string `json:"orderType"` // limit/market и т.п.
		TriggerPrice string `json:"triggerPrice"`
		ExecutePrice string `json:"executePrice"` // может быть "0" для market

		Size   string `json:"size"`
		Status string `json:"planStatus"` // not_trigger / triggered / canceled etc

		CTime string `json:"cTime"`
		UTime string `json:"uTime"`
	} `json:"entrustedList"`
}
