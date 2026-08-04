package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_ordersHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	user      string
	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64
	orderID   *string
}

func (s *futures_ordersHistory) Symbol(symbol string) *futures_ordersHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_ordersHistory) StartTime(startTime int64) *futures_ordersHistory {
	s.startTime = &startTime
	return s
}

func (s *futures_ordersHistory) EndTime(endTime int64) *futures_ordersHistory {
	s.endTime = &endTime
	return s
}

func (s *futures_ordersHistory) Limit(limit int64) *futures_ordersHistory {
	s.limit = &limit
	return s
}

func (s *futures_ordersHistory) Page(page int64) *futures_ordersHistory {
	s.page = &page
	return s
}

func (s *futures_ordersHistory) OrderID(orderID string) *futures_ordersHistory {
	s.orderID = &orderID
	return s
}

func (s *futures_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersHistory, err error) {
	if strings.TrimSpace(s.user) == "" {
		return nil, fmt.Errorf("hyperliquid futures ordersHistory: main user address is empty")
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

	out := s.convert.convertFuturesOrdersHistory(answ)

	if s.symbol != nil && strings.TrimSpace(*s.symbol) != "" {
		want := strings.TrimSpace(*s.symbol)
		filtered := make([]entity.Futures_OrdersHistory, 0, len(out))
		for _, it := range out {
			if futuresOrdersHistoryMatchSymbol(it.Symbol, want) {
				filtered = append(filtered, it)
			}
		}
		out = filtered
	}

	if s.orderID != nil && strings.TrimSpace(*s.orderID) != "" {
		want := strings.TrimSpace(*s.orderID)
		filtered := make([]entity.Futures_OrdersHistory, 0, len(out))
		for _, it := range out {
			if it.OrderID == want {
				filtered = append(filtered, it)
			}
		}
		out = filtered
	}

	if s.startTime != nil {
		filtered := make([]entity.Futures_OrdersHistory, 0, len(out))
		for _, it := range out {
			if it.CreateTime >= *s.startTime || it.UpdateTime >= *s.startTime {
				filtered = append(filtered, it)
			}
		}
		out = filtered
	}

	if s.endTime != nil {
		filtered := make([]entity.Futures_OrdersHistory, 0, len(out))
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
			return []entity.Futures_OrdersHistory{}, nil
		}
		end := start + limit
		if end > int64(len(out)) {
			end = int64(len(out))
		}
		out = out[start:end]
	}

	return out, nil
}

func futuresOrdersHistoryMatchSymbol(got string, want string) bool {
	g := strings.ToUpper(strings.TrimSpace(got))
	w := strings.ToUpper(strings.TrimSpace(want))
	if g == w {
		return true
	}
	if strings.TrimSuffix(g, "/USDC") == w {
		return true
	}
	if g == w+"/USDC" {
		return true
	}
	if strings.ReplaceAll(g, "/", "") == strings.ReplaceAll(w, "/", "") {
		return true
	}
	return false
}
