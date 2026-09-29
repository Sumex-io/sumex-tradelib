package xemus

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

	tpsl    bool
	tpOrder bool
	slOrder bool
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

// TpSl marks the id as a take-profit / stop-loss row's, i.e. an algo order id, as on cancelOrder.
func (s *futures_amendOrder) TpSl(v bool) *futures_amendOrder {
	s.tpsl = v
	return s
}

// TpOrder and SlOrder mark the id as a TP/SL row's and say which leg to change. One of them is
// needed when the order has both a take-profit and a stop-loss, since the two rows share an id.
func (s *futures_amendOrder) TpOrder(v bool) *futures_amendOrder {
	s.tpOrder = v
	return s
}

func (s *futures_amendOrder) SlOrder(v bool) *futures_amendOrder {
	s.slOrder = v
	return s
}

// Do edits an open order, by the kind of row it came from (see toAlgoOrders):
//   - a TP/SL row (TpSl, TpOrder or SlOrder set): one leg of the algo order, see amendTpSl;
//   - an algoOrderRef id: a bracket entry not yet placed, or a stop order, see amendAlgo;
//   - otherwise a resting regular order (PUT /v1/orders/:id). perp-api's edit takes the whole
//     order — symbol, type, side and quantity are required — so the order is read first
//     (GET /v1/orders/:id) and only the fields the caller changes are replaced. Orderly's quantity
//     on an edit is the order's total quantity, not what remains.
func (s *futures_amendOrder) Do(ctx context.Context) (res []entity.PlaceOrder, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return nil, errNoSymbol
	}
	symbol := strings.TrimSpace(*s.symbol)
	if s.orderID == nil {
		return nil, errors.New("xemus: amendOrder requires an orderID")
	}
	orderID, isAlgo, err := parseOrderRef(*s.orderID)
	if err != nil {
		return nil, err
	}
	if !hasValue(s.newSize) && !hasValue(s.newPrice) && !hasValue(s.newClientOrderID) {
		return nil, errors.New("xemus: amendOrder requires a new size, price or clientOrderID")
	}
	if s.tpOrder && s.slOrder {
		return nil, errors.New("xemus: an amend changes a take-profit or a stop-loss, not both")
	}
	if s.tpsl || s.tpOrder || s.slOrder {
		if isAlgo {
			return nil, fmt.Errorf("xemus: order %s is not a take-profit or stop-loss", strings.TrimSpace(*s.orderID))
		}
		return s.amendTpSl(ctx, symbol, orderID)
	}
	if isAlgo {
		return s.amendAlgo(ctx, symbol, orderID)
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
		return nil, fmt.Errorf("xemus: order %s is on %s, not %s", orderID, original.Symbol, symbol)
	}
	side := strings.ToUpper(strings.TrimSpace(original.Side))
	if s.side != nil && strings.TrimSpace(string(*s.side)) != "" && !strings.EqualFold(strings.TrimSpace(string(*s.side)), side) {
		return nil, fmt.Errorf("xemus: order %s is a %s order; an edit cannot change its side", orderID, side)
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
		return nil, fmt.Errorf("xemus: a %s order has no price to amend", orderType)
	}

	if original.ReduceOnly {
		body["reduce_only"] = true
	}
	clientOrderID := string(original.ClientOrderID)
	if hasValue(s.newClientOrderID) {
		clientOrderID = strings.TrimSpace(*s.newClientOrderID)
		if !clientOrderIDPattern.MatchString(clientOrderID) {
			return nil, fmt.Errorf("xemus: clientOrderID %q must be 1-36 characters of A-Z, a-z, 0-9, _ or -", clientOrderID)
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

// amendTpSl changes one TAKE_PROFIT / STOP_LOSS leg of the algo order the row's id names (for a
// bracket, the bracket root; its legs sit one level down). Orderly edits a leg through its parent
// algo order, addressing the leg by its own id — PUT /v1/algo-orders/:parent with
// child_orders [{order_id: leg, ...}], as Orderly's own UI does.
//
// The new price is the trigger price: the row shows the trigger level, and every leg this
// connector places closes at market. The new size is the quantity a TP_SL closes and is set on
// every one of its legs; a POSITIONAL_TP_SL closes the whole position and has no size.
func (s *futures_amendOrder) amendTpSl(ctx context.Context, symbol, orderID string) ([]entity.PlaceOrder, error) {
	if hasValue(s.newClientOrderID) {
		return nil, errors.New("xemus: a take-profit or stop-loss cannot be given a new clientOrderID")
	}
	root, err := s.getAlgoOrder(ctx, symbol, orderID)
	if err != nil {
		return nil, err
	}

	want := ""
	switch {
	case s.tpOrder:
		want = "TAKE_PROFIT"
	case s.slOrder:
		want = "STOP_LOSS"
	}
	var matches []openLeg
	for _, l := range openLegs(root) {
		if want == "" || strings.EqualFold(l.leg.AlgoType, want) {
			matches = append(matches, l)
		}
	}
	switch {
	case len(matches) == 0 && want == "":
		return nil, fmt.Errorf("xemus: order %s has no open take-profit or stop-loss", orderID)
	case len(matches) == 0:
		return nil, fmt.Errorf("xemus: order %s has no open %s", orderID, strings.ToLower(strings.ReplaceAll(want, "_", "-")))
	case len(matches) > 1:
		return nil, fmt.Errorf("xemus: order %s has both a take-profit and a stop-loss; set TpOrder or SlOrder to say which one to amend", orderID)
	}
	target := matches[0]
	if err := s.checkSide(orderID, target.leg.Side); err != nil {
		return nil, err
	}

	var children []map[string]interface{}
	if hasValue(s.newSize) {
		if !strings.EqualFold(target.parent.AlgoType, "TP_SL") {
			return nil, errors.New("xemus: a position take-profit / stop-loss closes the whole position and has no size to amend")
		}
		quantity, err := wireNumber(*s.newSize, "size")
		if err != nil {
			return nil, err
		}
		for _, l := range openLegs(target.parent) {
			children = append(children, map[string]interface{}{"order_id": json.Number(l.leg.AlgoOrderID.String()), "quantity": quantity})
		}
	}
	var leg map[string]interface{}
	for _, c := range children {
		if c["order_id"] == json.Number(target.leg.AlgoOrderID.String()) {
			leg = c
		}
	}
	if leg == nil {
		leg = map[string]interface{}{"order_id": json.Number(target.leg.AlgoOrderID.String())}
		children = append(children, leg)
	}
	if hasValue(s.newPrice) {
		if leg["trigger_price"], err = wireNumber(*s.newPrice, "trigger price"); err != nil {
			return nil, err
		}
	}

	path := "/v1/algo-orders/" + target.parent.AlgoOrderID.String()
	if _, err := s.callAPI(ctx, &request{method: http.MethodPut, path: path, body: map[string]interface{}{"child_orders": children}, signed: true}); err != nil {
		return nil, err
	}
	return []entity.PlaceOrder{{
		OrderID:       orderID,
		ClientOrderID: string(root.ClientOrderID),
		Ts:            time.Now().UnixMilli(),
	}}, nil
}

// amendAlgo edits an algo order listed under an algoOrderRef id (PUT /v1/algo-orders/:id): a
// bracket whose entry is not yet placed — its price and quantity — or a stop order — its trigger
// price, the level its row shows, and its quantity.
func (s *futures_amendOrder) amendAlgo(ctx context.Context, symbol, orderID string) ([]entity.PlaceOrder, error) {
	if hasValue(s.newClientOrderID) {
		return nil, errors.New("xemus: an algo order cannot be given a new clientOrderID")
	}
	root, err := s.getAlgoOrder(ctx, symbol, orderID)
	if err != nil {
		return nil, err
	}
	if err := s.checkSide(orderID, root.Side); err != nil {
		return nil, err
	}

	body := map[string]interface{}{}
	switch algoType := strings.ToUpper(root.AlgoType); algoType {
	case "BRACKET":
		if root.IsTriggered {
			return nil, fmt.Errorf("xemus: the entry of bracket %s has already been placed; amend its take-profit or stop-loss instead", orderID)
		}
		if hasValue(s.newPrice) {
			if strings.EqualFold(root.Type, "MARKET") {
				return nil, errors.New("xemus: a MARKET order has no price to amend")
			}
			if body["price"], err = wireNumber(*s.newPrice, "price"); err != nil {
				return nil, err
			}
		}
	case "STOP":
		if hasValue(s.newPrice) {
			if body["trigger_price"], err = wireNumber(*s.newPrice, "trigger price"); err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("xemus: a %s algo order cannot be amended here", algoType)
	}
	if hasValue(s.newSize) {
		if body["quantity"], err = wireNumber(*s.newSize, "size"); err != nil {
			return nil, err
		}
	}

	if _, err := s.callAPI(ctx, &request{method: http.MethodPut, path: "/v1/algo-orders/" + orderID, body: body, signed: true}); err != nil {
		return nil, err
	}
	return []entity.PlaceOrder{{
		OrderID:       algoOrderRef(orderID),
		ClientOrderID: string(root.ClientOrderID),
		Ts:            time.Now().UnixMilli(),
	}}, nil
}

func (s *futures_amendOrder) getAlgoOrder(ctx context.Context, symbol, orderID string) (algoOrderRow, error) {
	var root algoOrderRow
	data, err := s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/algo-orders/" + orderID, signed: true})
	if err != nil {
		return root, err
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return root, err
	}
	if root.Symbol != symbol {
		return root, fmt.Errorf("xemus: order %s is on %s, not %s", orderID, root.Symbol, symbol)
	}
	return root, nil
}

// checkSide refuses a side that disagrees with the order's own: an edit cannot flip it.
func (s *futures_amendOrder) checkSide(orderID, orderSide string) error {
	orderSide = strings.ToUpper(strings.TrimSpace(orderSide))
	if s.side != nil && strings.TrimSpace(string(*s.side)) != "" && orderSide != "" && !strings.EqualFold(strings.TrimSpace(string(*s.side)), orderSide) {
		return fmt.Errorf("xemus: order %s is a %s order; an edit cannot change its side", orderID, orderSide)
	}
	return nil
}

// openLeg is a TAKE_PROFIT / STOP_LOSS leg still waiting to trigger, with the algo order it
// belongs to.
type openLeg struct {
	leg, parent algoOrderRow
}

// openLegs collects the open legs anywhere below n: directly under a TP_SL / POSITIONAL_TP_SL,
// or one level further down under a bracket's POSITIONAL_TP_SL.
func openLegs(n algoOrderRow) []openLeg {
	var out []openLeg
	for _, c := range n.ChildOrders {
		switch strings.ToUpper(c.AlgoType) {
		case "TAKE_PROFIT", "STOP_LOSS":
			if !bool(c.IsTriggered) && c.isLive() {
				out = append(out, openLeg{leg: c, parent: n})
			}
		default:
			if !bool(c.IsTriggered) {
				out = append(out, openLegs(c)...)
			}
		}
	}
	return out
}
