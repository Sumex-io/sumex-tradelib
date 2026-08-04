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

// THE central fact governing every write action in this connector: the REST body and the EIP-712
// wallet signature encode the same type/side/timeInForce/selfTradePrevention/triggerType fields
// DIFFERENTLY. POST /v1/orders takes lowercase strings ("market", "buy", "gtc", "dc"); the signed
// `Order` struct takes the numeric enum as a uint8 (API_NOTES.md section B). Every resolution below
// derives BOTH encodings from a single call (resolveOrderType/resolveOrderTypeName,
// resolveOrderSide) so the string on the wire and the integer in the signature cannot describe two
// different orders. That is also why signing stays on the builder rather than moving into the
// transport as a RequestOption — see the note on katanaSigner in request.go.

type futures_placeOrder struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       futures_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)
	markets       func(ctx context.Context, opts ...utils.RequestOption) (map[string]Market, error)
	sign          *katanaSigner
	brokerID      string

	symbol        *string
	side          *entity.SideType
	size          *string
	price         *string
	orderType     *entity.OrderType
	clientOrderID *string

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

// OrderType takes the BASE type only — MARKET or LIMIT. Katana's six order types are the base type
// crossed with the TP/SL flags (see resolveOrderType), so a trigger order is requested as
// LIMIT/MARKET plus TpOrder/SlOrder, never as entity.OrderTypeStop or entity.OrderTypeTakeProfit.
// Those two are rejected rather than guessed at.
func (s *futures_placeOrder) OrderType(orderType entity.OrderType) *futures_placeOrder {
	s.orderType = &orderType
	return s
}

func (s *futures_placeOrder) ClientOrderID(clientOrderID string) *futures_placeOrder {
	s.clientOrderID = &clientOrderID
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

// Do builds a resolvedOrder from the request and submits it. Three decisions worth stating, since
// none are visible from the code alone:
//   - price is omitted from the wire and left at "0.00000000" in the signature for every *Market
//     variant, because Katana documents `price` as omitted for market orders.
//   - triggerType is always `last`. There is no input to select `index`; this is an assumption, not
//     sandbox-verified behaviour.
//   - reduceOnly is FORCED true for TP/SL regardless of the request — a stop that could increase
//     exposure is not a stop.
func (s *futures_placeOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return nil, errors.New("katana: placeOrder requires a symbol")
	}
	if s.side == nil || strings.TrimSpace(string(*s.side)) == "" {
		return nil, errors.New("katana: placeOrder requires a side")
	}
	if s.size == nil || strings.TrimSpace(*s.size) == "" {
		return nil, errors.New("katana: placeOrder requires a size")
	}
	symbol := strings.TrimSpace(*s.symbol)

	orderTypeInput := ""
	if s.orderType != nil {
		orderTypeInput = string(*s.orderType)
	}
	isTakeProfit := s.tpOrder != nil && *s.tpOrder
	isStopLoss := s.slOrder != nil && *s.slOrder

	sideCode, sideName, err := resolveOrderSide(string(*s.side))
	if err != nil {
		return nil, err
	}
	typeCode, err := resolveOrderType(orderTypeInput, isTakeProfit, isStopLoss)
	if err != nil {
		return nil, err
	}
	typeName, err := resolveOrderTypeName(orderTypeInput, isTakeProfit, isStopLoss)
	if err != nil {
		return nil, err
	}

	quantity, err := formatMoneyField("size", *s.size)
	if err != nil {
		return nil, err
	}

	isLimit := typeCode == orderTypeLimit || typeCode == orderTypeStopLossLimit || typeCode == orderTypeTakeProfitLimit
	limitPrice := "0.00000000"
	priceOnWire := ""
	if isLimit {
		if s.price == nil || strings.TrimSpace(*s.price) == "" {
			return nil, errors.New("katana: a limit order requires a price")
		}
		limitPrice, err = formatMoneyField("price", *s.price)
		if err != nil {
			return nil, err
		}
		priceOnWire = limitPrice
	}

	triggerPriceRaw := ""
	switch {
	case isTakeProfit:
		triggerPriceRaw = resolveTriggerPriceRaw(s.tpPrice, s.price)
		if triggerPriceRaw == "" {
			return nil, errors.New("katana: a take-profit order requires tpPrice or price")
		}
	case isStopLoss:
		triggerPriceRaw = resolveTriggerPriceRaw(s.slPrice, s.price)
		if triggerPriceRaw == "" {
			return nil, errors.New("katana: a stop-loss order requires slPrice or price")
		}
	}

	triggerPrice := "0.00000000"
	triggerPriceOnWire := ""
	triggerTypeCode := triggerTypeNone
	triggerTypeOnWire := ""
	if triggerPriceRaw != "" {
		triggerPrice, err = formatMoneyField("triggerPrice", triggerPriceRaw)
		if err != nil {
			return nil, err
		}
		triggerPriceOnWire = triggerPrice
		triggerTypeCode = triggerTypeLast
		triggerTypeOnWire = triggerTypeNameLast
	}

	reduceOnly := isTakeProfit || isStopLoss
	if !reduceOnly && s.reduce != nil {
		reduceOnly = *s.reduce
	}

	clientOrderID := ""
	if s.clientOrderID != nil {
		clientOrderID = strings.TrimSpace(*s.clientOrderID)
	}
	clientOrderID = resolveClientOrderID(s.brokerID, clientOrderID)

	raw, err := submitOrder(ctx, s.callAPI, s.resolveWallet, s.sign, resolvedOrder{
		MarketSymbol:       symbol,
		TypeCode:           typeCode,
		TypeName:           typeName,
		SideCode:           sideCode,
		SideName:           sideName,
		Quantity:           quantity,
		LimitPrice:         limitPrice,
		PriceOnWire:        priceOnWire,
		TriggerPrice:       triggerPrice,
		TriggerPriceOnWire: triggerPriceOnWire,
		TriggerTypeCode:    triggerTypeCode,
		TriggerTypeOnWire:  triggerTypeOnWire,
		ReduceOnly:         reduceOnly,
		// No inputs for either; GTC/DC are Katana's own documented REST defaults.
		TimeInForceCode:           timeInForceGTC,
		TimeInForceOnWire:         timeInForceNameGTC,
		SelfTradePreventionCode:   selfTradePreventionDC,
		SelfTradePreventionOnWire: selfTradePreventionNameDC,
		ClientOrderID:             clientOrderID,
	}, opts...)
	if err != nil {
		return nil, err
	}

	return s.convert.convertPlaceOrder(raw), nil
}

// ===============ORDER ENUMS=================
//
// Every enum below has two constant blocks: the uint8 signed into the EIP-712 struct, and the
// lowercase string sent on the REST wire (API_NOTES.md sections A and B). The resolve* functions
// are the only converters between this library's own upper-case inputs ("MARKET"/"BUY") and these
// two encodings, so nothing downstream juggles a bare uint8 or a magic string.

const (
	orderTypeMarket           uint8 = 0
	orderTypeLimit            uint8 = 1
	orderTypeStopLossMarket   uint8 = 2
	orderTypeStopLossLimit    uint8 = 3
	orderTypeTakeProfitMarket uint8 = 4
	orderTypeTakeProfitLimit  uint8 = 5
)

const (
	orderTypeNameMarket           = "market"
	orderTypeNameLimit            = "limit"
	orderTypeNameStopLossMarket   = "stopLossMarket"
	orderTypeNameStopLossLimit    = "stopLossLimit"
	orderTypeNameTakeProfitMarket = "takeProfitMarket"
	orderTypeNameTakeProfitLimit  = "takeProfitLimit"
)

const (
	orderSideBuy  uint8 = 0
	orderSideSell uint8 = 1
)

const (
	orderSideNameBuy  = "buy"
	orderSideNameSell = "sell"
)

const (
	triggerTypeNone  uint8 = 0
	triggerTypeLast  uint8 = 1
	triggerTypeIndex uint8 = 2
)

const (
	triggerTypeNameLast  = "last"
	triggerTypeNameIndex = "index"
)

const (
	timeInForceGTC uint8 = 0
	timeInForceGTX uint8 = 1
	timeInForceIOC uint8 = 2
	timeInForceFOK uint8 = 3
)

// placeOrder only ever sends GTC — it has no input to request otherwise. The other three exist for
// amendOrder, which carries an original order's policy through to its replacement
// (resolveTimeInForce).
const (
	timeInForceNameGTC = "gtc"
	timeInForceNameGTX = "gtx"
	timeInForceNameIOC = "ioc"
	timeInForceNameFOK = "fok"
)

const (
	selfTradePreventionDC uint8 = 0
	selfTradePreventionCO uint8 = 1
	selfTradePreventionCN uint8 = 2
	selfTradePreventionCB uint8 = 3
)

// CO/CN/CB exist for amendOrder, same as the non-GTC time-in-force names above.
const (
	selfTradePreventionNameDC = "dc"
	selfTradePreventionNameCO = "co"
	selfTradePreventionNameCN = "cn"
	selfTradePreventionNameCB = "cb"
)

// orderTypeNameByCode is the single bridge from the signed numeric enum to its REST-wire string.
// resolveOrderTypeName goes through this table rather than a second parallel switch, so the wire
// string and the signed integer cannot drift apart.
var orderTypeNameByCode = map[uint8]string{
	orderTypeMarket:           orderTypeNameMarket,
	orderTypeLimit:            orderTypeNameLimit,
	orderTypeStopLossMarket:   orderTypeNameStopLossMarket,
	orderTypeStopLossLimit:    orderTypeNameStopLossLimit,
	orderTypeTakeProfitMarket: orderTypeNameTakeProfitMarket,
	orderTypeTakeProfitLimit:  orderTypeNameTakeProfitLimit,
}

// resolveOrderType maps a base order type plus the tp/sl flags to the numeric Order Type Enum. An
// order can never be both take-profit and stop-loss; that is rejected rather than silently resolved
// in favour of one.
func resolveOrderType(orderType string, tp, sl bool) (uint8, error) {
	if tp && sl {
		return 0, errors.New("katana: an order cannot be both take-profit and stop-loss")
	}
	switch strings.ToUpper(strings.TrimSpace(orderType)) {
	case "MARKET":
		switch {
		case tp:
			return orderTypeTakeProfitMarket, nil
		case sl:
			return orderTypeStopLossMarket, nil
		default:
			return orderTypeMarket, nil
		}
	case "LIMIT":
		switch {
		case tp:
			return orderTypeTakeProfitLimit, nil
		case sl:
			return orderTypeStopLossLimit, nil
		default:
			return orderTypeLimit, nil
		}
	default:
		return 0, fmt.Errorf("katana: unknown order type %q (want MARKET or LIMIT)", orderType)
	}
}

// resolveOrderTypeName is resolveOrderType's REST-wire twin, derived FROM it rather than
// reimplemented, so body and signature cannot disagree about which of the six types was meant.
func resolveOrderTypeName(orderType string, tp, sl bool) (string, error) {
	code, err := resolveOrderType(orderType, tp, sl)
	if err != nil {
		return "", err
	}
	name, ok := orderTypeNameByCode[code]
	if !ok {
		return "", fmt.Errorf("katana: no REST name registered for order type code %d", code)
	}
	return name, nil
}

// resolveOrderSide returns both encodings at once, for the same no-drift reason as the type pair.
func resolveOrderSide(side string) (uint8, string, error) {
	switch strings.ToUpper(strings.TrimSpace(side)) {
	case "BUY":
		return orderSideBuy, orderSideNameBuy, nil
	case "SELL":
		return orderSideSell, orderSideNameSell, nil
	default:
		return 0, "", fmt.Errorf("katana: unknown order side %q (want BUY or SELL)", side)
	}
}

// resolveTriggerPriceRaw prefers the dedicated tpPrice/slPrice input but FALLS BACK to the order's
// own price, and the fallback is the load-bearing half: callers commonly build a TP/SL as
// {orderType: LIMIT, price, tpOrder: true} and set no dedicated field at all — so requiring it
// would reject every Katana TP/SL before any network call and leave the position unprotected. A
// blank string counts as absent; these callers set optional strings rather than omitting them.
func resolveTriggerPriceRaw(dedicated, price *string) string {
	for _, candidate := range []*string{dedicated, price} {
		if candidate == nil {
			continue
		}
		if trimmed := strings.TrimSpace(*candidate); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// formatMoneyField re-renders a positive decimal as a zero-padded 8-decimal string — the wire
// contract every quantity, price and margin value in this connector follows.
func formatMoneyField(label, value string) (string, error) {
	f, err := parsePositiveDecimal(value, label)
	if err != nil {
		return "", err
	}
	return formatDecimal8(f), nil
}

// maxClientOrderIDBytes is Katana's documented `clientOrderId` limit.
const maxClientOrderIDBytes = 40

// truncateClientOrderID keeps an id inside the cap, so a builder-code prefix or a generated
// replacement id cannot push it over and get the whole order rejected.
func truncateClientOrderID(id string) string {
	if len(id) <= maxClientOrderIDBytes {
		return id
	}
	return id[:maxClientOrderIDBytes]
}

// resolveClientOrderID prefixes the broker/builder code (which is how Katana attributes builder
// rewards) and truncates to the clientOrderId cap. Unset means no prefix.
func resolveClientOrderID(brokerID, base string) string {
	clientOrderID := base
	if builder := strings.TrimSpace(brokerID); builder != "" {
		clientOrderID = builder + clientOrderID
	}
	return truncateClientOrderID(clientOrderID)
}

// ===============SIGNED REQUEST ENVELOPE=================

// katanaSignedRequest is the envelope every POST/DELETE Trade request uses: parameters under a
// top-level object, with the EIP-712 wallet signature as its sibling.
type katanaSignedRequest[T any] struct {
	Parameters T      `json:"parameters"`
	Signature  string `json:"signature"`
}

// katanaPlaceOrderParams is the POST /v1/orders `parameters` object. Every enum-shaped field here
// carries the lowercase REST STRING form; the numeric encoding of the same order lives only in the
// separately-signed orderSignPayload. See the file-level comment.
type katanaPlaceOrderParams struct {
	Nonce               string `json:"nonce"`
	Wallet              string `json:"wallet"`
	DelegatedKey        string `json:"delegatedKey,omitempty"`
	Market              string `json:"market"`
	Type                string `json:"type"`
	Side                string `json:"side"`
	Quantity            string `json:"quantity"`
	Price               string `json:"price,omitempty"`
	TriggerPrice        string `json:"triggerPrice,omitempty"`
	TriggerType         string `json:"triggerType,omitempty"`
	ClientOrderId       string `json:"clientOrderId,omitempty"`
	ReduceOnly          bool   `json:"reduceOnly"`
	TimeInForce         string `json:"timeInForce,omitempty"`
	SelfTradePrevention string `json:"selfTradePrevention,omitempty"`
}

// resolvedOrder carries an order's fully-resolved fields — both encodings of every enum and the
// formatted decimals — so submitOrder never re-derives anything from a caller-facing type.
// placeOrder builds one from its inputs; amendOrder builds one from the ORIGINAL order on the book,
// which is the only available source for the fields an amend request omits.
type resolvedOrder struct {
	MarketSymbol       string
	TypeCode           uint8
	TypeName           string
	SideCode           uint8
	SideName           string
	Quantity           string
	LimitPrice         string // signed value; "0.00000000" for a *Market variant
	PriceOnWire        string // wire value; "" (omitted) for a *Market variant
	TriggerPrice       string // signed value; "0.00000000" when there is no trigger
	TriggerPriceOnWire string // wire value; "" (omitted) when there is no trigger
	TriggerTypeCode    uint8
	TriggerTypeOnWire  string // "" (omitted) when there is no trigger
	ReduceOnly         bool
	// amendOrder carries the ORIGINAL order's values through here, so a resting post-only (gtx)
	// order does not silently become GTC — and therefore taker-fillable — the moment its price is
	// edited. placeOrder has no input for either and always sends GTC/DC.
	TimeInForceCode           uint8
	TimeInForceOnWire         string
	SelfTradePreventionCode   uint8
	SelfTradePreventionOnWire string
	ClientOrderID             string
}

// submitOrder signs and posts a fully-resolved order — the shared tail end of placeOrder and
// amendOrder's replacement. It resolves the wallet itself, AFTER its caller has fully validated the
// order, so a request that can never be valid costs no network call.
//
// DelegatedPublicKey is passed explicitly rather than left empty so that orderTypedData's mismatch
// guard, which only runs on a non-empty value, actually defends the body-versus-signature invariant
// in production instead of only under unit test. Both sides come from the same derivation, so this
// cannot introduce a mismatch of its own.
//
// The ErrorCode check is not redundant with the HTTP status: Katana returns 200 with
// errorCode/errorMessage set when the matching engine force-cancels an order it accepted
// (QUANTITY_TOO_LOW, REDUCE_ONLY_POSITION_CLOSED, SELF_TRADE, TIME_IN_FORCE) — without it a
// fully-rejected order looks like a successful placement with nothing on the book. A non-zero
// executedQuantity means a real fill happened before the cancellation, which IS a successful order,
// so only the zero-fill case is an error.
func submitOrder(
	ctx context.Context,
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error),
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error),
	sign *katanaSigner,
	o resolvedOrder,
	opts ...utils.RequestOption,
) (katanaOrder, error) {
	wallet, err := resolveWallet(ctx, opts...)
	if err != nil {
		return katanaOrder{}, err
	}
	delegated, err := sign.delegatedAddress()
	if err != nil {
		return katanaOrder{}, err
	}
	nonceStr, nonceBig, err := newNonce()
	if err != nil {
		return katanaOrder{}, err
	}

	sig, err := sign.signOrder(orderSignPayload{
		Nonce:               nonceBig,
		Wallet:              wallet,
		MarketSymbol:        o.MarketSymbol,
		OrderType:           o.TypeCode,
		OrderSide:           o.SideCode,
		Quantity:            o.Quantity,
		LimitPrice:          o.LimitPrice,
		TriggerPrice:        o.TriggerPrice,
		TriggerType:         o.TriggerTypeCode,
		IsReduceOnly:        o.ReduceOnly,
		TimeInForce:         o.TimeInForceCode,
		SelfTradePrevention: o.SelfTradePreventionCode,
		DelegatedPublicKey:  delegated,
		ClientOrderId:       o.ClientOrderID,
	})
	if err != nil {
		return katanaOrder{}, err
	}

	body := katanaSignedRequest[katanaPlaceOrderParams]{
		Parameters: katanaPlaceOrderParams{
			Nonce:               nonceStr,
			Wallet:              wallet,
			DelegatedKey:        delegated,
			Market:              o.MarketSymbol,
			Type:                o.TypeName,
			Side:                o.SideName,
			Quantity:            o.Quantity,
			Price:               o.PriceOnWire,
			TriggerPrice:        o.TriggerPriceOnWire,
			TriggerType:         o.TriggerTypeOnWire,
			ClientOrderId:       o.ClientOrderID,
			ReduceOnly:          o.ReduceOnly,
			TimeInForce:         o.TimeInForceOnWire,
			SelfTradePrevention: o.SelfTradePreventionOnWire,
		},
		Signature: sig,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return katanaOrder{}, err
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/v1/orders",
		SecType:  utils.SecTypeSigned,
	}
	r.BodyString = string(b)

	data, _, err := callAPI(ctx, r, opts...)
	if err != nil {
		return katanaOrder{}, err
	}

	answ := katanaOrder{}
	if err := json.Unmarshal(data, &answ); err != nil {
		return katanaOrder{}, err
	}
	if answ.ErrorCode != "" && isZeroDecimal(answ.ExecutedQuantity) {
		return katanaOrder{}, fmt.Errorf("katana: order rejected: %s: %s", answ.ErrorCode, answ.ErrorMessage)
	}

	return answ, nil
}
