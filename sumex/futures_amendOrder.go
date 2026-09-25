package sumex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

type futures_amendOrder struct {
	callAPI callAPIFunc

	symbol           *string
	side             *entity.SideType
	orderID          *string
	newSize          *string
	newPrice         *string
	newClientOrderID *string
}

func (s *futures_amendOrder) Symbol(symbol string) *futures_amendOrder {
	s.symbol = &symbol
	return s
}

// Side is accepted for parity with the other connectors. An edit cannot flip an order's side, so a
// side that disagrees with the order's own is refused rather than applied.
func (s *futures_amendOrder) Side(side entity.SideType) *futures_amendOrder {
	s.side = &side
	return s
}

func (s *futures_amendOrder) OrderID(orderID string) *futures_amendOrder {
	s.orderID = &orderID
	return s
}

func (s *futures_amendOrder) NewSize(newSize string) *futures_amendOrder {
	s.newSize = &newSize
	return s
}

func (s *futures_amendOrder) NewPrice(newPrice string) *futures_amendOrder {
	s.newPrice = &newPrice
	return s
}

func (s *futures_amendOrder) NewClientOrderID(newClientOrderID string) *futures_amendOrder {
	s.newClientOrderID = &newClientOrderID
	return s
}

// Do edits a resting regular order (PUT /v1/orders/:id). perp-api's edit takes the whole order —
// symbol, type, side and quantity are required — so the order is read first
// (GET /v1/orders/:id) and only the fields the caller changes are replaced. Orderly's quantity on
// an edit is the order's total quantity, not what remains. TP/SL (algo) orders are not editable
// through this action.
func (s *futures_amendOrder) Do(ctx context.Context) (res []entity.PlaceOrder, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return nil, errNoSymbol
	}
	symbol := strings.TrimSpace(*s.symbol)
	if s.orderID == nil || !orderIDPattern.MatchString(strings.TrimSpace(*s.orderID)) {
		return nil, errors.New("sumex: amendOrder requires a numeric orderID")
	}
	orderID := strings.TrimSpace(*s.orderID)
	if !hasValue(s.newSize) && !hasValue(s.newPrice) && !hasValue(s.newClientOrderID) {
		return nil, errors.New("sumex: amendOrder requires a new size, price or clientOrderID")
	}

	data, err := s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/orders/" + orderID, signed: true})
	if err != nil {
		return nil, err
	}
	var original orderRow
	if err := json.Unmarshal(data, &original); err != nil {
		return nil, err
	}
	if original.Symbol != symbol {
		return nil, fmt.Errorf("sumex: order %s is on %s, not %s", orderID, original.Symbol, symbol)
	}
	side := strings.ToUpper(strings.TrimSpace(original.Side))
	if s.side != nil && strings.TrimSpace(string(*s.side)) != "" && !strings.EqualFold(strings.TrimSpace(string(*s.side)), side) {
		return nil, fmt.Errorf("sumex: order %s is a %s order; an edit cannot change its side", orderID, side)
	}
	orderType := strings.ToUpper(strings.TrimSpace(original.Type))

	body := map[string]interface{}{
		"symbol":     symbol,
		"order_type": orderType,
		"side":       side,
	}

	quantity := original.Quantity.String()
	if hasValue(s.newSize) {
		quantity = *s.newSize
	}
	if body["order_quantity"], err = wireNumber(quantity, "size"); err != nil {
		return nil, err
	}

	// perp-api refuses a price on MARKET / ASK / BID and requires one on every other type.
	if orderType != "MARKET" && orderType != "ASK" && orderType != "BID" {
		price := original.Price.String()
		if hasValue(s.newPrice) {
			price = *s.newPrice
		}
		if body["order_price"], err = wireNumber(price, "price"); err != nil {
			return nil, err
		}
	} else if hasValue(s.newPrice) {
		return nil, fmt.Errorf("sumex: a %s order has no price to amend", orderType)
	}

	if original.ReduceOnly {
		body["reduce_only"] = true
	}
	clientOrderID := string(original.ClientOrderID)
	if hasValue(s.newClientOrderID) {
		clientOrderID = strings.TrimSpace(*s.newClientOrderID)
		if !clientOrderIDPattern.MatchString(clientOrderID) {
			return nil, fmt.Errorf("sumex: clientOrderID %q must be 1-36 characters of A-Z, a-z, 0-9, _ or -", clientOrderID)
		}
		body["client_order_id"] = clientOrderID
	}

	if _, err := s.callAPI(ctx, &request{method: http.MethodPut, path: "/v1/orders/" + orderID, body: body, signed: true}); err != nil {
		return nil, err
	}
	return []entity.PlaceOrder{{
		OrderID:       orderID,
		ClientOrderID: clientOrderID,
		Ts:            time.Now().UnixMilli(),
	}}, nil
}
