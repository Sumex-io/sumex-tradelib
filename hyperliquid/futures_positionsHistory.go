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

type futures_positionsHistory struct {
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

func (s *futures_positionsHistory) Symbol(symbol string) *futures_positionsHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_positionsHistory) StartTime(startTime int64) *futures_positionsHistory {
	s.startTime = &startTime
	return s
}

func (s *futures_positionsHistory) EndTime(endTime int64) *futures_positionsHistory {
	s.endTime = &endTime
	return s
}

func (s *futures_positionsHistory) Limit(limit int64) *futures_positionsHistory {
	s.limit = &limit
	return s
}

func (s *futures_positionsHistory) Page(page int64) *futures_positionsHistory {
	s.page = &page
	return s
}

func (s *futures_positionsHistory) OrderID(orderID string) *futures_positionsHistory {
	s.orderID = &orderID
	return s
}

func (s *futures_positionsHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_PositionsHistory, err error) {
	if strings.TrimSpace(s.user) == "" {
		return nil, fmt.Errorf("hyperliquid futures positionsHistory: main user address is empty")
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/info",
		SecType:  utils.SecTypeNone,
	}

	payload := map[string]interface{}{
		"user": s.user,
	}

	if s.startTime != nil || s.endTime != nil {
		payload["type"] = "userFillsByTime"
		if s.startTime != nil {
			payload["startTime"] = *s.startTime
		}
		if s.endTime != nil {
			payload["endTime"] = *s.endTime
		}
	} else {
		payload["type"] = "userFills"
	}

	b, _ := json.Marshal(payload)
	r.BodyString = string(b)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	var answ []hlUserFill
	if err := json.Unmarshal(data, &answ); err != nil {
		return nil, err
	}

	out := s.convert.convertPositionsHistory(answ)

	if s.symbol != nil && strings.TrimSpace(*s.symbol) != "" {
		want := strings.TrimSpace(*s.symbol)
		filtered := make([]entity.Futures_PositionsHistory, 0, len(out))
		for _, it := range out {
			if futuresOrdersHistoryMatchSymbol(it.Symbol, want) {
				filtered = append(filtered, it)
			}
		}
		out = filtered
	}

	if s.orderID != nil && strings.TrimSpace(*s.orderID) != "" {
		// want := strings.TrimSpace(*s.orderID)
		// filtered := make([]entity.Futures_PositionsHistory, 0, len(out))
		// for _, it := range out {
		// 	if it.OrderID == want {
		// 		filtered = append(filtered, it)
		// 	}
		// }
		// out = filtered
	}

	if s.startTime != nil {
		filtered := make([]entity.Futures_PositionsHistory, 0, len(out))
		for _, it := range out {
			if it.UpdateTime >= *s.startTime {
				filtered = append(filtered, it)
			}
		}
		out = filtered
	}

	if s.endTime != nil {
		filtered := make([]entity.Futures_PositionsHistory, 0, len(out))
		for _, it := range out {
			if it.UpdateTime <= *s.endTime {
				filtered = append(filtered, it)
			}
		}
		out = filtered
	}

	sort.SliceStable(out, func(i, j int) bool {
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
			return []entity.Futures_PositionsHistory{}, nil
		}
		end := start + limit
		if end > int64(len(out)) {
			end = int64(len(out))
		}
		out = out[start:end]
	}

	return out, nil
}

type hlUserFill struct {
	Coin          string      `json:"coin"`
	Px            string      `json:"px"`
	Sz            string      `json:"sz"`
	Side          string      `json:"side"`
	Time          int64       `json:"time"`
	StartPosition string      `json:"startPosition"`
	Dir           string      `json:"dir"`
	ClosedPnl     string      `json:"closedPnl"`
	Hash          string      `json:"hash"`
	Oid           interface{} `json:"oid"`
	Crossed       bool        `json:"crossed"`
	Fee           string      `json:"fee"`
	BuilderFee    string      `json:"builderFee"`
	Tid           interface{} `json:"tid"`
	FeeToken      string      `json:"feeToken"`
}
