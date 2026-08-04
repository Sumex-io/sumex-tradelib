package katana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_amendOrder struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       futures_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)
	markets       func(ctx context.Context, opts ...utils.RequestOption) (map[string]Market, error)
	sign          *katanaSigner
	brokerID      string

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

// Do implements Katana's non-existent native amend as cancel-then-place, mirroring how the
// hyperliquid connector amends. The replacement is built and validated FIRST, so the only remaining
// window between cancel and place is submitOrder's own network call.
//
// There is deliberately no rollback if that window loses: by then the original no longer exists,
// Katana has no undo-cancel, and its place in the book cannot be reconstructed. The place error is
// instead wrapped with the cancelled order's id so the caller can see unambiguously that the
// original is gone and no replacement exists, rather than reading a generic failure as "nothing
// happened".
func (s *futures_amendOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return nil, errors.New("katana: amendOrder requires a symbol")
	}
	if s.orderID == nil || strings.TrimSpace(*s.orderID) == "" {
		return nil, errors.New("katana: amendOrder requires an orderID")
	}
	// `side` and `newSize` are deliberately NOT required: a chart's drag-to-amend path omits both,
	// and buildAmendReplacement defaults them from the original order.
	if s.newPrice == nil || strings.TrimSpace(*s.newPrice) == "" {
		return nil, errors.New("katana: amendOrder requires newPrice")
	}
	symbol := strings.TrimSpace(*s.symbol)
	orderID := strings.TrimSpace(*s.orderID)

	wallet, err := s.resolveWallet(ctx, opts...)
	if err != nil {
		return nil, err
	}

	original, err := fetchOrderByID(ctx, s.callAPI, wallet, orderID, opts...)
	if err != nil {
		return nil, fmt.Errorf("katana: amendOrder could not read order %s before cancelling it: %w", orderID, err)
	}

	replacement, err := s.buildAmendReplacement(symbol, orderID, original)
	if err != nil {
		return nil, err
	}

	cancel := &futures_cancelOrder{
		callAPI:       s.callAPI,
		convert:       s.convert,
		resolveWallet: s.resolveWallet,
		sign:          s.sign,
	}
	if _, err := cancel.OrderID(orderID).Do(ctx, opts...); err != nil {
		return nil, fmt.Errorf("katana: amendOrder could not cancel order %s, no replacement was placed: %w", orderID, err)
	}

	raw, err := submitOrder(ctx, s.callAPI, s.resolveWallet, s.sign, replacement, opts...)
	if err != nil {
		return nil, fmt.Errorf("katana: amendOrder canceled order %s but the replacement order failed to place — the position is now unprotected by that order: %w", orderID, err)
	}

	return s.convert.convertPlaceOrder(raw), nil
}

// buildAmendReplacement resolves and FULLY validates the replacement order before Do cancels
// anything. That ordering is the whole point of it being a separate function: triggerType and
// triggerPrice are optional response fields, so validating after the cancel would let an original
// that omits one be cancelled and only then fail, leaving the position unprotected by any order at
// all. It makes no network call — newNonce() is pure.
//
// Everything the amend request cannot express is carried from the original order: market, side,
// quantity, reduceOnly, trigger fields, timeInForce and selfTradePrevention. Market and side are
// additionally cross-checked against the caller's values, so a stale UI row cannot cancel a resting
// BUY and place a SELL in its place on what the user experienced as a price edit.
func (s *futures_amendOrder) buildAmendReplacement(symbol, orderID string, original katanaOrder) (resolvedOrder, error) {
	if !strings.EqualFold(strings.TrimSpace(original.Market), symbol) {
		return resolvedOrder{}, fmt.Errorf("katana: amendOrder: request symbol %q does not match order %s's actual market %q", symbol, orderID, original.Market)
	}

	// Side and NewSize are optional and a chart's drag-to-amend sends neither, so both default from
	// the original. NewSize is deliberately not cross-checked — changing the size is the point of
	// supplying it.
	originalSideCode, originalSideName, err := resolveOrderSide(original.Side)
	if err != nil {
		return resolvedOrder{}, fmt.Errorf("katana: amendOrder: order %s's own side %q is unusable: %w", orderID, original.Side, err)
	}
	sideCode, sideName := originalSideCode, originalSideName
	if s.side != nil && strings.TrimSpace(string(*s.side)) != "" {
		requestedCode, requestedName, serr := resolveOrderSide(string(*s.side))
		if serr != nil {
			return resolvedOrder{}, serr
		}
		if requestedCode != originalSideCode {
			return resolvedOrder{}, fmt.Errorf("katana: amendOrder: request side %q does not match order %s's actual side %q", requestedName, orderID, originalSideName)
		}
		sideCode, sideName = requestedCode, requestedName
	}

	quantitySource, quantityLabel := original.OriginalQuantity, "originalQuantity"
	if s.newSize != nil && strings.TrimSpace(*s.newSize) != "" {
		quantitySource, quantityLabel = *s.newSize, "newSize"
	}
	quantity, err := formatMoneyField(quantityLabel, quantitySource)
	if err != nil {
		return resolvedOrder{}, err
	}
	limitPrice, err := formatMoneyField("newPrice", *s.newPrice)
	if err != nil {
		return resolvedOrder{}, err
	}

	isTakeProfit, isStopLoss := classifyTriggerOrder(original.Type)
	typeCode, err := resolveOrderType(orderTypeNameToPlaceOrderInput, isTakeProfit, isStopLoss)
	if err != nil {
		return resolvedOrder{}, err
	}
	typeName, err := resolveOrderTypeName(orderTypeNameToPlaceOrderInput, isTakeProfit, isStopLoss)
	if err != nil {
		return resolvedOrder{}, err
	}

	triggerPrice := "0.00000000"
	triggerPriceOnWire := ""
	triggerTypeCode := triggerTypeNone
	triggerTypeOnWire := ""
	if isTakeProfit || isStopLoss {
		if strings.TrimSpace(original.TriggerPrice) == "" {
			return resolvedOrder{}, fmt.Errorf("katana: amendOrder: order %s is classified take-profit/stop-loss but carries no triggerPrice to preserve", orderID)
		}
		triggerPrice, err = formatMoneyField("triggerPrice", original.TriggerPrice)
		if err != nil {
			return resolvedOrder{}, err
		}
		triggerPriceOnWire = triggerPrice
		triggerTypeCode, triggerTypeOnWire, err = resolveTriggerType(original.TriggerType)
		if err != nil {
			return resolvedOrder{}, err
		}
	}

	timeInForceCode, timeInForceOnWire, err := resolveTimeInForce(original.TimeInForce)
	if err != nil {
		return resolvedOrder{}, err
	}
	selfTradePreventionCode, selfTradePreventionOnWire, err := resolveSelfTradePrevention(original.SelfTradePrevention)
	if err != nil {
		return resolvedOrder{}, err
	}

	clientOrderID := ""
	if s.newClientOrderID != nil {
		clientOrderID = strings.TrimSpace(*s.newClientOrderID)
	}
	if clientOrderID == "" {
		generated, _, err := newNonce()
		if err != nil {
			return resolvedOrder{}, err
		}
		clientOrderID = generated
	}
	clientOrderID = resolveClientOrderID(s.brokerID, clientOrderID)

	return resolvedOrder{
		MarketSymbol:              original.Market,
		TypeCode:                  typeCode,
		TypeName:                  typeName,
		SideCode:                  sideCode,
		SideName:                  sideName,
		Quantity:                  quantity,
		LimitPrice:                limitPrice,
		PriceOnWire:               limitPrice,
		TriggerPrice:              triggerPrice,
		TriggerPriceOnWire:        triggerPriceOnWire,
		TriggerTypeCode:           triggerTypeCode,
		TriggerTypeOnWire:         triggerTypeOnWire,
		ReduceOnly:                original.ReduceOnly,
		TimeInForceCode:           timeInForceCode,
		TimeInForceOnWire:         timeInForceOnWire,
		SelfTradePreventionCode:   selfTradePreventionCode,
		SelfTradePreventionOnWire: selfTradePreventionOnWire,
		ClientOrderID:             clientOrderID,
	}, nil
}

// An amend supplies a new size/price but never a new base type, so the replacement is always LIMIT
// — the only type a price parameter unambiguously applies to, and what the hyperliquid connector
// hardcodes for the same reason. The TP/SL classification and trigger fields still come from the
// original.
const orderTypeNameToPlaceOrderInput = "LIMIT"

// fetchOrderByID reads the original order amendOrder must preserve fields from. When `orderId` is
// supplied Katana ignores every other filter, so only nonce/wallet/orderId are sent. A missing
// order, or more than one row for a single id, is a hard error — an amend must not proceed past
// either.
func fetchOrderByID(
	ctx context.Context,
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error),
	wallet, orderID string,
	opts ...utils.RequestOption,
) (katanaOrder, error) {
	nonce, _, err := newNonce()
	if err != nil {
		return katanaOrder{}, err
	}

	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v1/orders",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(utils.Params{
		"nonce":   nonce,
		"wallet":  wallet,
		"orderId": orderID,
	})

	data, _, err := callAPI(ctx, r, opts...)
	if err != nil {
		return katanaOrder{}, err
	}

	var raw []katanaOrder
	if err := json.Unmarshal(data, &raw); err != nil {
		return katanaOrder{}, err
	}
	if len(raw) == 0 {
		return katanaOrder{}, fmt.Errorf("katana: order %s not found", orderID)
	}
	if len(raw) > 1 {
		return katanaOrder{}, fmt.Errorf("katana: GET /v1/orders?orderId=%s returned %d rows, want at most 1", orderID, len(raw))
	}
	return raw[0], nil
}

// resolveTriggerType is the wire-string-to-enum direction, used by amendOrder to carry an original
// order's own triggerType through instead of re-deriving it as `last`.
func resolveTriggerType(triggerType string) (uint8, string, error) {
	switch strings.ToLower(strings.TrimSpace(triggerType)) {
	case triggerTypeNameLast:
		return triggerTypeLast, triggerTypeNameLast, nil
	case triggerTypeNameIndex:
		return triggerTypeIndex, triggerTypeNameIndex, nil
	default:
		return 0, "", fmt.Errorf("katana: unknown trigger type %q (want last or index)", triggerType)
	}
}

// resolveTimeInForce maps a wire timeInForce back to its enum, so an amend keeps a resting GTX order
// post-only instead of quietly making it taker-fillable. Empty input falls back to GTC rather than
// erroring: the field is documented present "only for limit orders", so a *Market original — which
// an amend may legally follow with a LIMIT replacement — can genuinely omit it.
func resolveTimeInForce(tif string) (uint8, string, error) {
	switch strings.ToLower(strings.TrimSpace(tif)) {
	case "", timeInForceNameGTC:
		return timeInForceGTC, timeInForceNameGTC, nil
	case timeInForceNameGTX:
		return timeInForceGTX, timeInForceNameGTX, nil
	case timeInForceNameIOC:
		return timeInForceIOC, timeInForceNameIOC, nil
	case timeInForceNameFOK:
		return timeInForceFOK, timeInForceNameFOK, nil
	default:
		return 0, "", fmt.Errorf("katana: unknown timeInForce %q", tif)
	}
}

// resolveSelfTradePrevention is resolveTimeInForce's twin. This field is documented as always
// present, but empty still falls back to DC rather than erroring — an amend must not fail because a
// documented guarantee did not hold.
func resolveSelfTradePrevention(stp string) (uint8, string, error) {
	switch strings.ToLower(strings.TrimSpace(stp)) {
	case "", selfTradePreventionNameDC:
		return selfTradePreventionDC, selfTradePreventionNameDC, nil
	case selfTradePreventionNameCO:
		return selfTradePreventionCO, selfTradePreventionNameCO, nil
	case selfTradePreventionNameCN:
		return selfTradePreventionCN, selfTradePreventionNameCN, nil
	case selfTradePreventionNameCB:
		return selfTradePreventionCB, selfTradePreventionNameCB, nil
	default:
		return 0, "", fmt.Errorf("katana: unknown selfTradePrevention %q", stp)
	}
}
