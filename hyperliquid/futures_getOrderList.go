package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	user      string
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
	if strings.TrimSpace(s.user) == "" {
		return nil, fmt.Errorf("hyperliquid futures getOrderList: main user address is empty")
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/info",
		SecType:  utils.SecTypeNone,
	}

	b, _ := json.Marshal(map[string]interface{}{
		"type": "frontendOpenOrders",
		"user": s.user,
	})
	r.BodyString = string(b)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}

	var answ []hlFrontendOpenOrderFutures
	if err := json.Unmarshal(data, &answ); err != nil {
		return nil, err
	}

	out := s.convert.convertFuturesOrderList(answ)

	if s.symbol != nil && strings.TrimSpace(*s.symbol) != "" {
		want := strings.TrimSpace(*s.symbol)
		filtered := make([]entity.Futures_OrdersList, 0, len(out))
		for _, it := range out {
			if futuresOrdersHistoryMatchSymbol(it.Symbol, want) {
				filtered = append(filtered, it)
			}
		}
		out = filtered
	}

	if s.orderType != nil {
		want := strings.ToUpper(strings.TrimSpace(string(*s.orderType)))
		filtered := make([]entity.Futures_OrdersList, 0, len(out))
		for _, it := range out {
			if strings.ToUpper(strings.TrimSpace(it.Type)) == want {
				filtered = append(filtered, it)
			}
		}
		out = filtered
	}

	if s.limit != nil && *s.limit > 0 && len(out) > *s.limit {
		out = out[:*s.limit]
	}

	return out, nil
}

type hlFrontendOpenOrderFutures struct {
	Coin       string `json:"coin"`
	Side       string `json:"side"`
	LimitPx    string `json:"limitPx"`
	Sz         string `json:"sz"`
	OrigSz     string `json:"origSz"`
	Oid        int64  `json:"oid"`
	Timestamp  int64  `json:"timestamp"`
	OrderType  string `json:"orderType"`
	Tif        string `json:"tif"`
	ReduceOnly bool   `json:"reduceOnly"`
	Cloid      string `json:"cloid"`
	C          string `json:"c"`

	// важно для TP/SL
	IsTrigger        bool   `json:"isTrigger"`
	TriggerCondition string `json:"triggerCondition"`
	TriggerPx        string `json:"triggerPx"`
	IsPositionTpsl   bool   `json:"isPositionTpsl"`
}
