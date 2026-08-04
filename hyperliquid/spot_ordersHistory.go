package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_ordersHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	user      string
	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64
}

func (s *spot_ordersHistory) Symbol(symbol string) *spot_ordersHistory {
	s.symbol = &symbol
	return s
}

func (s *spot_ordersHistory) StartTime(startTime int64) *spot_ordersHistory {
	s.startTime = &startTime
	return s
}

func (s *spot_ordersHistory) EndTime(endTime int64) *spot_ordersHistory {
	s.endTime = &endTime
	return s
}

func (s *spot_ordersHistory) Limit(limit int64) *spot_ordersHistory {
	s.limit = &limit
	return s
}

func (s *spot_ordersHistory) Page(page int64) *spot_ordersHistory {
	s.page = &page
	return s
}

func (s *spot_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_OrdersHistory, err error) {
	if strings.TrimSpace(s.user) == "" {
		return nil, fmt.Errorf("hyperliquid spot ordersHistory: main user address is empty")
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/info",
		SecType:  utils.SecTypeNone,
	}

	b, _ := json.Marshal(map[string]interface{}{
		"type": "historicalOrders",
		"user": s.user,
	})
	r.BodyString = string(b)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	var answ []hlHistoricalOrder
	if err := json.Unmarshal(data, &answ); err != nil {
		return nil, err
	}

	out := s.convert.convertSpotOrdersHistory(answ)

	if s.symbol != nil && strings.TrimSpace(*s.symbol) != "" {
		want := strings.TrimSpace(*s.symbol)
		filtered := make([]entity.Spot_OrdersHistory, 0, len(out))
		for _, it := range out {
			if it.Symbol == want {
				filtered = append(filtered, it)
			}
		}
		out = filtered
	}

	if s.startTime != nil {
		filtered := make([]entity.Spot_OrdersHistory, 0, len(out))
		for _, it := range out {
			if it.CreateTime >= *s.startTime || it.UpdateTime >= *s.startTime {
				filtered = append(filtered, it)
			}
		}
		out = filtered
	}

	if s.endTime != nil {
		filtered := make([]entity.Spot_OrdersHistory, 0, len(out))
		for _, it := range out {
			if it.CreateTime <= *s.endTime || it.UpdateTime <= *s.endTime {
				filtered = append(filtered, it)
			}
		}
		out = filtered
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UpdateTime == out[j].UpdateTime {
			return out[i].CreateTime > out[j].CreateTime
		}
		return out[i].UpdateTime > out[j].UpdateTime
	})

	if s.limit != nil && *s.limit > 0 {
		page := int64(1)
		if s.page != nil && *s.page > 0 {
			page = *s.page
		}

		limit := *s.limit
		start := (page - 1) * limit
		if start >= int64(len(out)) {
			return []entity.Spot_OrdersHistory{}, nil
		}
		end := start + limit
		if end > int64(len(out)) {
			end = int64(len(out))
		}
		out = out[start:end]
	}

	return out, nil
}

type hlHistoricalOrder struct {
	Order struct {
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

		TriggerCondition string        `json:"triggerCondition"`
		IsTrigger        bool          `json:"isTrigger"`
		TriggerPx        string        `json:"triggerPx"`
		IsPositionTpsl   bool          `json:"isPositionTpsl"`
		Children         []interface{} `json:"children"`
	} `json:"order"`

	Status          string `json:"status"`
	StatusTimestamp int64  `json:"statusTimestamp"`
}

func (o hlHistoricalOrder) clientOrderID() string {
	if strings.TrimSpace(o.Order.Cloid) != "" {
		return strings.TrimSpace(o.Order.Cloid)
	}
	if strings.TrimSpace(o.Order.C) != "" {
		return strings.TrimSpace(o.Order.C)
	}
	return ""
}

func parseSpotAssetIDFromHistoricalCoin(coin string) (assetID string, ok bool) {
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
