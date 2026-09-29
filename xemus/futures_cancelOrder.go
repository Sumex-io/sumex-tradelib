package xemus

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

// algoOrderIDPrefix marks an order id from Orderly's algo-order sequence on a row that is not a
// TP/SL: a bracket entry still waiting to be placed, or a stop order. Such a row carries no
// TpOrder/SlOrder flag for the platform to send back, so the id itself has to say which route
// cancels or edits it. TP/SL rows keep the bare numeric id they have always had and are routed by
// the flag.
const algoOrderIDPrefix = "algo-"

func algoOrderRef(id string) string { return algoOrderIDPrefix + id }

// parseOrderRef splits an id the platform sends back into Orderly's numeric id and whether it
// names an algo order.
func parseOrderRef(raw string) (id string, algo bool, err error) {
	raw = strings.TrimSpace(raw)
	id = strings.TrimPrefix(raw, algoOrderIDPrefix)
	if !orderIDPattern.MatchString(id) {
		return "", false, fmt.Errorf("xemus: orderID %q is not an order id", raw)
	}
	return id, id != raw, nil
}

// Do cancels a regular order (DELETE /v1/orders/:id) or an algo order (DELETE /v1/algo-orders/:id)
// — a TP/SL (TpSl set), or any id carrying algoOrderIDPrefix. There is deliberately no fallback
// from one to the other: regular and algo order ids are separate sequences on Orderly, so retrying
// a missing regular id as an algo id could cancel an unrelated order that happens to share the
// number.
func (s *futures_cancelOrder) Do(ctx context.Context) (res []entity.PlaceOrder, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return nil, errNoSymbol
	}
	if s.orderID == nil {
		return nil, errors.New("xemus: cancelOrder requires an orderID")
	}
	orderID, isAlgo, err := parseOrderRef(*s.orderID)
	if err != nil {
		return nil, err
	}

	path := "/v1/orders/" + orderID
	if s.tpsl || isAlgo {
		path = "/v1/algo-orders/" + orderID
	}
	q := url.Values{}
	q.Set("symbol", strings.TrimSpace(*s.symbol))

	if _, err := s.callAPI(ctx, &request{method: http.MethodDelete, path: path, query: q, signed: true}); err != nil {
		return nil, err
	}
	return []entity.PlaceOrder{{OrderID: strings.TrimSpace(*s.orderID), Ts: time.Now().UnixMilli()}}, nil
}
