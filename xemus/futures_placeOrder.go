package xemus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

type futures_placeOrder struct {
	callAPI callAPIFunc

	symbol        *string
	side          *entity.SideType
	size          *string
	price         *string
	orderType     *entity.OrderType
	clientOrderID *string
	marginMode    *string

	reduce  *bool
	tpOrder *bool
	slOrder *bool
	tpPrice *string
	slPrice *string
}

func (s *futures_placeOrder) Symbol(symbol string) *futures_placeOrder {
	s.symbol = &symbol
	return s
}

func (s *futures_placeOrder) Side(side entity.SideType) *futures_placeOrder {
	s.side = &side
	return s
}

func (s *futures_placeOrder) Size(size string) *futures_placeOrder {
	s.size = &size
	return s
}

func (s *futures_placeOrder) Price(price string) *futures_placeOrder {
	s.price = &price
	return s
}

// OrderType takes MARKET or LIMIT, or one of Orderly's limit variants IOC, FOK and POST_ONLY. A
// trigger order is requested as TpOrder/SlOrder, never as STOP or TAKE_PROFIT.
func (s *futures_placeOrder) OrderType(orderType entity.OrderType) *futures_placeOrder {
	s.orderType = &orderType
	return s
}

func (s *futures_placeOrder) ClientOrderID(clientOrderID string) *futures_placeOrder {
	s.clientOrderID = &clientOrderID
	return s
}

// MarginMode places the order in CROSS or ISOLATED margin. Unset, Orderly uses the symbol's default
// margin mode (see NewSetMarginMode).
func (s *futures_placeOrder) MarginMode(marginMode string) *futures_placeOrder {
	s.marginMode = &marginMode
	return s
}

func (s *futures_placeOrder) Reduce(reduce bool) *futures_placeOrder {
	s.reduce = &reduce
	return s
}

func (s *futures_placeOrder) TpOrder(v bool) *futures_placeOrder {
	s.tpOrder = &v
	return s
}

func (s *futures_placeOrder) SlOrder(v bool) *futures_placeOrder {
	s.slOrder = &v
	return s
}

func (s *futures_placeOrder) TpPrice(tpPrice string) *futures_placeOrder {
	s.tpPrice = &tpPrice
	return s
}

func (s *futures_placeOrder) SlPrice(slPrice string) *futures_placeOrder {
	s.slPrice = &slPrice
	return s
}

// perp-api's client_order_id pattern. Checked here so a bad id fails with a clear message instead
// of a VALIDATION_ERROR path.
var clientOrderIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,36}$`)

// Do places one of three things:
//   - a regular order (POST /v1/orders);
//   - with TpOrder or SlOrder set, a take-profit or stop-loss on the open position
//     (POST /v1/algo-orders, a single-leg TP_SL);
//   - with TpPrice and/or SlPrice on an entry order, a BRACKET (POST /v1/algo-orders): the entry
//     plus a POSITIONAL_TP_SL that Orderly activates once the entry fills. Its legs close the WHOLE
//     position, not only what this entry adds — the same as a TP/SL attached on Bybit or OKX in
//     their default full-position mode.
//
// A bracket is an algo order until its entry is placed, so the id it returns — and the id its
// open-order row carries — is algoOrderRef(algo id).
func (s *futures_placeOrder) Do(ctx context.Context) (res []entity.PlaceOrder, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return nil, errNoSymbol
	}
	if s.side == nil {
		return nil, errors.New("xemus: placeOrder requires a side")
	}
	side := strings.ToUpper(strings.TrimSpace(string(*s.side)))
	if side != "BUY" && side != "SELL" {
		return nil, fmt.Errorf("xemus: unknown side %q (want BUY or SELL)", side)
	}
	if s.size == nil {
		return nil, errors.New("xemus: placeOrder requires a size")
	}
	quantity, err := wireNumber(*s.size, "size")
	if err != nil {
		return nil, err
	}

	clientOrderID := ""
	if s.clientOrderID != nil {
		clientOrderID = strings.TrimSpace(*s.clientOrderID)
		if clientOrderID != "" && !clientOrderIDPattern.MatchString(clientOrderID) {
			return nil, fmt.Errorf("xemus: clientOrderID %q must be 1-36 characters of A-Z, a-z, 0-9, _ or -", clientOrderID)
		}
	}
	marginMode := ""
	if s.marginMode != nil && strings.TrimSpace(*s.marginMode) != "" {
		if marginMode, err = wireMarginMode(*s.marginMode); err != nil {
			return nil, err
		}
	}

	isTakeProfit := s.tpOrder != nil && *s.tpOrder
	isStopLoss := s.slOrder != nil && *s.slOrder
	if isTakeProfit && isStopLoss {
		return nil, errors.New("xemus: an order cannot be both take-profit and stop-loss")
	}

	body := map[string]interface{}{"symbol": strings.TrimSpace(*s.symbol)}
	if clientOrderID != "" {
		body["client_order_id"] = clientOrderID
	}
	if marginMode != "" {
		body["margin_mode"] = marginMode
	}

	path := "/v1/orders"
	isBracket := false
	switch {
	case isTakeProfit || isStopLoss:
		path = "/v1/algo-orders"
		leg, err := s.triggerLeg(side, isTakeProfit)
		if err != nil {
			return nil, err
		}
		body["algo_type"] = "TP_SL"
		body["quantity"] = quantity
		body["child_orders"] = []interface{}{leg}
	case hasValue(s.tpPrice) || hasValue(s.slPrice):
		path, isBracket = "/v1/algo-orders", true
		if err := s.bracket(body, side, quantity); err != nil {
			return nil, err
		}
	default:
		orderType, err := s.regularOrderType()
		if err != nil {
			return nil, err
		}
		body["order_type"] = orderType
		body["side"] = side
		body["order_quantity"] = quantity
		if orderType != "MARKET" {
			if !hasValue(s.price) {
				return nil, fmt.Errorf("xemus: a %s order requires a price", orderType)
			}
			price, err := wireNumber(*s.price, "price")
			if err != nil {
				return nil, err
			}
			body["order_price"] = price
		}
		if s.reduce != nil && *s.reduce {
			body["reduce_only"] = true
		}
	}

	data, err := s.callAPI(ctx, &request{method: http.MethodPost, path: path, body: body, signed: true})
	if err != nil {
		return nil, err
	}
	var answ placeOrderResponse
	if err := json.Unmarshal(data, &answ); err != nil {
		return nil, err
	}
	resolvedClientOrderID := string(answ.ClientOrderID)
	if resolvedClientOrderID == "" {
		resolvedClientOrderID = clientOrderID
	}
	orderID := answ.OrderID.String()
	if isBracket {
		orderID = algoOrderRef(orderID)
	}
	return []entity.PlaceOrder{{
		OrderID:       orderID,
		ClientOrderID: resolvedClientOrderID,
		Ts:            time.Now().UnixMilli(),
	}}, nil
}

// triggerLeg builds the single TAKE_PROFIT / STOP_LOSS leg. The trigger level is tpPrice/slPrice,
// falling back to price — the platform's set-TP/SL path sends the level as `price`.
//
// The leg always closes at MARKET once triggered, whatever order type the caller sent: the
// platform sends LIMIT for every TP/SL, and a stop-loss that rests as a limit at its trigger price
// can be jumped by a fast market and never fill. perp-api forces reduce_only on every leg.
func (s *futures_placeOrder) triggerLeg(side string, isTakeProfit bool) (map[string]interface{}, error) {
	algoType, dedicated := "STOP_LOSS", s.slPrice
	if isTakeProfit {
		algoType, dedicated = "TAKE_PROFIT", s.tpPrice
	}
	raw := s.price
	if hasValue(dedicated) {
		raw = dedicated
	}
	if !hasValue(raw) {
		return nil, fmt.Errorf("xemus: a %s order requires a trigger price", strings.ToLower(strings.ReplaceAll(algoType, "_", "-")))
	}
	trigger, err := wireNumber(*raw, "trigger price")
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"algo_type":     algoType,
		"side":          side,
		"type":          "MARKET",
		"trigger_price": trigger,
	}, nil
}

// bracket fills body with a BRACKET: the entry, and one POSITIONAL_TP_SL group holding a
// TAKE_PROFIT and/or STOP_LOSS leg on the closing side. Each leg closes at market
// (CLOSE_POSITION) once triggered, for the reason given on triggerLeg. perp-api checks that the
// legs sit on the right sides of each other and of a LIMIT entry's price.
func (s *futures_placeOrder) bracket(body map[string]interface{}, side string, quantity json.Number) error {
	orderType, err := s.regularOrderType()
	if err != nil {
		return err
	}
	// Orderly's bracket entry is MARKET or LIMIT only.
	if orderType != "MARKET" && orderType != "LIMIT" {
		return fmt.Errorf("xemus: a take-profit or stop-loss can only be attached to a MARKET or LIMIT order, not %s", orderType)
	}
	// perp-api's bracket schema has no reduce_only and would strip it, turning a closing order
	// into an opening one.
	if s.reduce != nil && *s.reduce {
		return errors.New("xemus: a take-profit or stop-loss cannot be attached to a reduce-only order")
	}

	body["algo_type"] = "BRACKET"
	body["side"] = side
	body["type"] = orderType
	body["quantity"] = quantity
	if orderType == "LIMIT" {
		if !hasValue(s.price) {
			return errors.New("xemus: a LIMIT order requires a price")
		}
		price, err := wireNumber(*s.price, "price")
		if err != nil {
			return err
		}
		body["price"] = price
	}

	closingSide := "SELL"
	if side == "SELL" {
		closingSide = "BUY"
	}
	var legs []interface{}
	for _, l := range []struct {
		algoType, label string
		price           *string
	}{
		{"TAKE_PROFIT", "take-profit price", s.tpPrice},
		{"STOP_LOSS", "stop-loss price", s.slPrice},
	} {
		if !hasValue(l.price) {
			continue
		}
		trigger, err := wireNumber(*l.price, l.label)
		if err != nil {
			return err
		}
		legs = append(legs, map[string]interface{}{
			"algo_type":     l.algoType,
			"side":          closingSide,
			"type":          "CLOSE_POSITION",
			"trigger_price": trigger,
		})
	}
	body["child_orders"] = []interface{}{map[string]interface{}{
		"algo_type":    "POSITIONAL_TP_SL",
		"child_orders": legs,
	}}
	return nil
}

func (s *futures_placeOrder) regularOrderType() (string, error) {
	if s.orderType == nil || strings.TrimSpace(string(*s.orderType)) == "" {
		return "", errors.New("xemus: placeOrder requires an order type")
	}
	switch t := strings.ToUpper(strings.TrimSpace(string(*s.orderType))); t {
	case "MARKET", "LIMIT", "IOC", "FOK", "POST_ONLY":
		return t, nil
	default:
		return "", fmt.Errorf("xemus: unsupported order type %q (want MARKET, LIMIT, IOC, FOK or POST_ONLY; use TpOrder/SlOrder for a trigger order)", t)
	}
}

func hasValue(s *string) bool {
	return s != nil && strings.TrimSpace(*s) != ""
}
