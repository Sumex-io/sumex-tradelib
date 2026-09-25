package sumex

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

type futures_cancelOrder struct {
	callAPI callAPIFunc

	symbol  *string
	orderID *string
	tpsl    bool
}

func (s *futures_cancelOrder) Symbol(symbol string) *futures_cancelOrder {
	s.symbol = &symbol
	return s
}

func (s *futures_cancelOrder) OrderID(orderID string) *futures_cancelOrder {
	s.orderID = &orderID
	return s
}

// TpSl marks the id as a take-profit / stop-loss, i.e. an algo order id.
func (s *futures_cancelOrder) TpSl(v bool) *futures_cancelOrder {
	s.tpsl = v
	return s
}

var orderIDPattern = regexp.MustCompile(`^[1-9][0-9]*$`)

// Do cancels a regular order (DELETE /v1/orders/:id) or, for a TP/SL, the algo order
// (DELETE /v1/algo-orders/:id). There is deliberately no fallback from one to the other: regular
// and algo order ids are separate sequences on Orderly, so retrying a missing regular id as an algo
// id could cancel an unrelated order that happens to share the number.
func (s *futures_cancelOrder) Do(ctx context.Context) (res []entity.PlaceOrder, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return nil, errNoSymbol
	}
	if s.orderID == nil {
		return nil, errors.New("sumex: cancelOrder requires an orderID")
	}
	orderID := strings.TrimSpace(*s.orderID)
	if !orderIDPattern.MatchString(orderID) {
		return nil, fmt.Errorf("sumex: orderID %q is not an order id", orderID)
	}

	path := "/v1/orders/" + orderID
	if s.tpsl {
		path = "/v1/algo-orders/" + orderID
	}
	q := url.Values{}
	q.Set("symbol", strings.TrimSpace(*s.symbol))

	if _, err := s.callAPI(ctx, &request{method: http.MethodDelete, path: path, query: q, signed: true}); err != nil {
		return nil, err
	}
	return []entity.PlaceOrder{{OrderID: orderID, Ts: time.Now().UnixMilli()}}, nil
}
