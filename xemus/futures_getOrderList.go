package xemus

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

// maxPageSize is the largest `size` a paginated list takes. perp-api accepts 500, but Orderly
// behind it answers a larger one with "size must be less than or equal to 100".
const maxPageSize = 100

type futures_getOrderList struct {
	callAPI callAPIFunc

	symbol *string
}

func (s *futures_getOrderList) Symbol(symbol string) *futures_getOrderList {
	s.symbol = &symbol
	return s
}

// Do merges the two places an open order lives on Orderly: resting regular orders
// (GET /v1/orders?status=INCOMPLETE) and untriggered algo orders — every TP/SL this connector
// places — (GET /v1/algo-orders?status=INCOMPLETE). Leaving the second out would render a
// protected position as unprotected. Both reads are one full page; a full page is logged rather
// than paged through, since a wallet with 100 resting orders is not a case the UI serves.
func (s *futures_getOrderList) Do(ctx context.Context) (res []entity.Futures_OrdersList, err error) {
	query := func() url.Values {
		q := url.Values{}
		q.Set("status", "INCOMPLETE")
		q.Set("size", strconv.Itoa(maxPageSize))
		if s.symbol != nil && strings.TrimSpace(*s.symbol) != "" {
			q.Set("symbol", strings.TrimSpace(*s.symbol))
		}
		return q
	}

	data, err := s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/orders", query: query(), signed: true})
	if err != nil {
		return res, err
	}
	var orders ordersResponse
	if err := json.Unmarshal(data, &orders); err != nil {
		return res, err
	}

	data, err = s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/algo-orders", query: query(), signed: true})
	if err != nil {
		return res, err
	}
	var algos algoOrdersResponse
	if err := json.Unmarshal(data, &algos); err != nil {
		return res, err
	}

	if len(orders.Rows) >= maxPageSize || len(algos.Rows) >= maxPageSize {
		log.Printf("xemus: open orders hit the %d-row page cap (orders=%d algo=%d); further rows are NOT in this response", maxPageSize, len(orders.Rows), len(algos.Rows))
	}

	res = make([]entity.Futures_OrdersList, 0, len(orders.Rows)+len(algos.Rows))
	for _, o := range orders.Rows {
		res = append(res, toOrder(o))
	}
	for _, a := range algos.Rows {
		res = append(res, toAlgoOrders(a)...)
	}
	return res, nil
}
