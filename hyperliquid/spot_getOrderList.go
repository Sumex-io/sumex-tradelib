package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	user   string
	symbol *string
	limit  *int
}

func (s *spot_getOrderList) Symbol(symbol string) *spot_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *spot_getOrderList) Limit(limit int) *spot_getOrderList {
	s.limit = &limit
	return s
}

func (s *spot_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_OrdersList, err error) {
	if strings.TrimSpace(s.user) == "" {
		return nil, fmt.Errorf("hyperliquid spot getOrderList: main user address is empty")
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
	var answ []hlFrontendOpenOrder
	if err := json.Unmarshal(data, &answ); err != nil {
		return nil, err
	}

	out := s.convert.convertSpotOpenOrders(answ)

	if s.symbol != nil && strings.TrimSpace(*s.symbol) != "" {
		want := strings.TrimSpace(*s.symbol)
		filtered := make([]entity.Spot_OrdersList, 0, len(out))
		for _, it := range out {
			if it.Symbol == want {
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

type hlFrontendOpenOrder struct {
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
}

func (o hlFrontendOpenOrder) clientOrderID() string {
	if strings.TrimSpace(o.Cloid) != "" {
		return strings.TrimSpace(o.Cloid)
	}
	if strings.TrimSpace(o.C) != "" {
		return strings.TrimSpace(o.C)
	}
	return ""
}

func parseSpotAssetIDFromCoin(coin string) (assetID string, ok bool) {
	c := strings.TrimSpace(coin)
	if strings.HasPrefix(c, "@") {
		n, err := strconv.Atoi(strings.TrimPrefix(c, "@"))
		if err != nil || n < 0 {
			return "", false
		}
		return strconv.Itoa(10000 + n), true
	}
	return "", false
}
