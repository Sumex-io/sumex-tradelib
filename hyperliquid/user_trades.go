package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/utils"
)

type hyperliquidUserTradesParams struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)

	user            string
	symbol          *string
	orderID         *string
	startTime       *int64
	endTime         *int64
	limit           *int64
	page            *int64
	aggregateByTime *bool
	errPrefix       string
}

func requestHyperliquidUserFills(ctx context.Context, p hyperliquidUserTradesParams, opts ...utils.RequestOption) ([]hlUserFill, error) {
	if strings.TrimSpace(p.user) == "" {
		return nil, fmt.Errorf("%s: main user address is empty", p.errPrefix)
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/info",
		SecType:  utils.SecTypeNone,
	}

	payload := map[string]interface{}{
		"user": p.user,
	}
	if p.startTime != nil || p.endTime != nil {
		payload["type"] = "userFillsByTime"
		if p.startTime == nil {
			return nil, fmt.Errorf("%s: startTime is required when using time filters", p.errPrefix)
		}
		payload["startTime"] = *p.startTime
		if p.endTime != nil {
			payload["endTime"] = *p.endTime
		}
	} else {
		payload["type"] = "userFills"
	}
	if p.aggregateByTime != nil {
		payload["aggregateByTime"] = *p.aggregateByTime
	}

	b, _ := json.Marshal(payload)
	r.BodyString = string(b)

	data, _, err := p.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}

	var fills []hlUserFill
	if err := json.Unmarshal(data, &fills); err != nil {
		return nil, err
	}

	if p.orderID != nil && strings.TrimSpace(*p.orderID) != "" {
		want := strings.TrimSpace(*p.orderID)
		filtered := make([]hlUserFill, 0, len(fills))
		for _, item := range fills {
			if stringifyHLID(item.Oid) == want {
				filtered = append(filtered, item)
			}
		}
		fills = filtered
	}

	if p.startTime != nil {
		filtered := make([]hlUserFill, 0, len(fills))
		for _, item := range fills {
			if item.Time >= *p.startTime {
				filtered = append(filtered, item)
			}
		}
		fills = filtered
	}
	if p.endTime != nil {
		filtered := make([]hlUserFill, 0, len(fills))
		for _, item := range fills {
			if item.Time <= *p.endTime {
				filtered = append(filtered, item)
			}
		}
		fills = filtered
	}

	sort.SliceStable(fills, func(i, j int) bool { return fills[i].Time > fills[j].Time })

	return fills, nil
}
