package sumex

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

// ===============DECIMALS=================

// Decimal arithmetic goes through big.Rat so no value picks up float64 noise on its way to a
// string field.

func parseRat(s string) (*big.Rat, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	return new(big.Rat).SetString(s)
}

// ratString renders r as a plain decimal with at most 18 fractional digits, trailing zeros
// trimmed.
func ratString(r *big.Rat) string {
	return trimTrailingZeros(r.FloatString(18))
}

func trimTrailingZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	s = strings.TrimSuffix(s, ".")
	if s == "-0" {
		return "0"
	}
	return s
}

// plainDecimal rewrites exponent notation ("1e-7") as a plain decimal and leaves anything else,
// including the empty string, as it is.
func plainDecimal(s string) string {
	if !strings.ContainsAny(s, "eE") {
		return s
	}
	r, ok := parseRat(s)
	if !ok {
		return s
	}
	return ratString(r)
}

func absDecimal(s string) string {
	r, ok := parseRat(s)
	if !ok {
		return s
	}
	return ratString(new(big.Rat).Abs(r))
}

func isNegative(s string) bool {
	r, ok := parseRat(s)
	return ok && r.Sign() < 0
}

func isZero(s string) bool {
	r, ok := parseRat(s)
	return !ok || r.Sign() == 0
}

// decimalPlaces turns a tick ("0.01", "1e-7") into the count of fractional digits the entity's
// *Precision fields carry.
func decimalPlaces(tick string) string {
	s := plainDecimal(strings.TrimSpace(tick))
	dot := strings.Index(s, ".")
	if dot < 0 {
		return "0"
	}
	return fmt.Sprintf("%d", len(strings.TrimRight(s[dot+1:], "0")))
}

// unrealizedPnl is Orderly's own definition: position_qty * (mark_price - average_open_price).
// The signed quantity makes it correct for shorts without a branch.
func unrealizedPnl(qty, mark, entry string) string {
	q, ok1 := parseRat(qty)
	m, ok2 := parseRat(mark)
	e, ok3 := parseRat(entry)
	if !ok1 || !ok2 || !ok3 {
		return ""
	}
	return ratString(new(big.Rat).Mul(q, new(big.Rat).Sub(m, e)))
}

func mulAbs(a, b string) string {
	x, ok1 := parseRat(a)
	y, ok2 := parseRat(b)
	if !ok1 || !ok2 {
		return ""
	}
	return ratString(new(big.Rat).Abs(new(big.Rat).Mul(x, y)))
}

func sub(a, b string) string {
	x, ok1 := parseRat(a)
	y, ok2 := parseRat(b)
	if !ok1 || !ok2 {
		return ""
	}
	return ratString(new(big.Rat).Sub(x, y))
}

// maxLeverageFromIMR is floor(1 / base_imr): the leverage a market allows at its base tier.
func maxLeverageFromIMR(imr string) string {
	r, ok := parseRat(imr)
	if !ok || r.Sign() <= 0 {
		return ""
	}
	inv := new(big.Rat).Inv(r)
	return new(big.Int).Quo(inv.Num(), inv.Denom()).String()
}

var plainPositiveDecimal = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

// wireNumber validates a caller-supplied decimal string and returns it as a json.Number, so it goes
// out as the JSON number perp-api requires (it rejects numeric strings) without passing through
// float64. Validation matters: json.Number marshals its text verbatim.
func wireNumber(s, label string) (json.Number, error) {
	s = strings.TrimSpace(s)
	if !plainPositiveDecimal.MatchString(s) {
		return "", fmt.Errorf("sumex: %s %q is not a plain decimal", label, s)
	}
	r, _ := parseRat(s)
	if r.Sign() <= 0 {
		return "", fmt.Errorf("sumex: %s must be greater than zero", label)
	}
	return json.Number(s), nil
}

// wireLeverage converts a leverage string to the integer perp-api takes (1..100). "10" and "10.0"
// are accepted; a fractional leverage is refused rather than silently truncated.
func wireLeverage(s string) (int64, error) {
	r, ok := parseRat(s)
	if !ok || !r.IsInt() {
		return 0, fmt.Errorf("sumex: leverage %q must be a whole number", s)
	}
	v := r.Num()
	if !v.IsInt64() || v.Int64() < 1 || v.Int64() > 100 {
		return 0, fmt.Errorf("sumex: leverage %q must be between 1 and 100", s)
	}
	return v.Int64(), nil
}

// ===============ROWS=================

// decodeRows accepts both shapes perp-api passes through for list endpoints whose response it does
// not validate: a bare array, or Orderly's {rows: [...]}.
func decodeRows(data []byte, out interface{}) error {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") {
		return json.Unmarshal(data, out)
	}
	var wrapper struct {
		Rows json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return err
	}
	if len(wrapper.Rows) == 0 || string(wrapper.Rows) == "null" {
		return nil
	}
	return json.Unmarshal(wrapper.Rows, out)
}

// splitSymbol reads base and quote out of Orderly's PERP_<BASE>_<QUOTE> symbol.
func splitSymbol(symbol string) (base, quote string, err error) {
	parts := strings.Split(symbol, "_")
	if len(parts) != 3 || parts[0] != "PERP" || parts[1] == "" || parts[2] == "" {
		return "", "", fmt.Errorf("sumex: unexpected symbol %q", symbol)
	}
	return parts[1], parts[2], nil
}

func normalizeMarginMode(m string) string {
	switch strings.ToUpper(strings.TrimSpace(m)) {
	case "ISOLATED", "1":
		return "ISOLATED"
	default:
		// Orderly's default mode, and the only one an account had before isolated margin existed.
		return "CROSS"
	}
}

// normalizeOrderType maps Orderly's order types onto the platform OrderType enum. IOC, FOK and
// POST_ONLY are limit orders with a time-in-force; ASK and BID are limit orders at the best price.
// Anything unknown degrades to LIMIT rather than passing through: one unmappable value fails the
// caller's output validation for every row in the response.
func normalizeOrderType(t string) string {
	switch strings.ToUpper(strings.TrimSpace(t)) {
	case "MARKET":
		return "MARKET"
	case "LIMIT", "IOC", "FOK", "POST_ONLY", "ASK", "BID":
		return "LIMIT"
	default:
		log.Printf("sumex: unknown order type %q reported as LIMIT", t)
		return "LIMIT"
	}
}

// triggerOrderType names a trigger order in the platform enum: the limit variant is STOP /
// TAKE_PROFIT, the market variant STOP_MARKET / TAKE_PROFIT_MARKET (the Binance naming the enum
// follows). CLOSE_POSITION, the type Orderly gives positional TP/SL legs, closes at market.
func triggerOrderType(isTakeProfit bool, orderType string) string {
	limit := strings.EqualFold(strings.TrimSpace(orderType), "LIMIT")
	switch {
	case isTakeProfit && limit:
		return "TAKE_PROFIT"
	case isTakeProfit:
		return "TAKE_PROFIT_MARKET"
	case limit:
		return "STOP"
	default:
		return "STOP_MARKET"
	}
}

// derivePositionSide infers the position an order acts on from its side. A closing order —
// reduce-only, or any TP/SL — targets the side opposite to an opening order of the same side;
// without the inversion a reduce-only SELL closing a long renders as "Open Short".
func derivePositionSide(side string, closing bool) string {
	if closing {
		if side == "SELL" {
			return "LONG"
		}
		return "SHORT"
	}
	if side == "SELL" {
		return "SHORT"
	}
	return "LONG"
}

// executedQuantity prefers total_executed_quantity and falls back to executed: Orderly documents
// both, and the checked perp-api response schema only guarantees `executed`.
func executedQuantity(o orderRow) string {
	if o.TotalExecutedQty != "" {
		return o.TotalExecutedQty.String()
	}
	return o.Executed.String()
}

func toInstrumentInfo(s symbolRow) (entity.Futures_InstrumentsInfo, error) {
	base, quote, err := splitSymbol(s.Symbol)
	if err != nil {
		return entity.Futures_InstrumentsInfo{}, err
	}
	// Consumers gate on state === "LIVE". A missing status predates Orderly's per-symbol trading
	// status and means tradable; REDUCE_ONLY / POST_ONLY / DELISTING markets keep their own state
	// so the pair sync skips them while positions on them still resolve.
	state := strings.ToUpper(strings.TrimSpace(s.Status))
	if state == "" || state == "ACTIVE" {
		state = "LIVE"
	}
	return entity.Futures_InstrumentsInfo{
		Symbol:         s.Symbol,
		Base:           base,
		Quote:          quote,
		MinQty:         s.BaseMin.String(),
		MinNotional:    s.MinNotional.String(),
		PricePrecision: decimalPlaces(s.QuoteTick.String()),
		SizePrecision:  decimalPlaces(s.BaseTick.String()),
		State:          state,
		MaxLeverage:    maxLeverageFromIMR(s.BaseIMR.String()),
		Multiplier:     "1",
		ContractSize:   "1",
		IsSizeContract: false,
	}, nil
}

// toBalance follows the library convention: Balance excludes unrealized PnL, Equity includes it.
// Orderly's total_collateral_value already carries unsettled PnL, so it is the equity, and the
// balance is what remains once the open positions' unrealized PnL is taken out.
func toBalance(p positionsResponse) []entity.FuturesBalance {
	unrealized := new(big.Rat)
	for _, row := range p.Rows {
		if u, ok := parseRat(unrealizedPnl(row.PositionQty.String(), row.MarkPrice.String(), row.AverageOpenPrice.String())); ok {
			unrealized.Add(unrealized, u)
		}
	}
	equity := p.TotalCollateralValue.String()
	upnl := ratString(unrealized)
	return []entity.FuturesBalance{
		{
			Asset:            "USDC",
			Balance:          sub(equity, upnl),
			Equity:           equity,
			Available:        p.FreeCollateral.String(),
			UnrealizedProfit: upnl,
		},
	}
}

func toPosition(p positionRow) entity.Futures_Positions {
	qty := p.PositionQty.String()
	side := "LONG"
	if isNegative(qty) {
		side = "SHORT"
	}
	ts := p.Timestamp.int64()
	return entity.Futures_Positions{
		Symbol:       p.Symbol,
		PositionSide: side,
		PositionSize: absDecimal(qty),
		Leverage:     p.Leverage.String(),
		// Orderly nets one position per symbol and has no id for an open one.
		PositionID:       "",
		EntryPrice:       p.AverageOpenPrice.String(),
		MarkPrice:        p.MarkPrice.String(),
		UnRealizedProfit: unrealizedPnl(qty, p.MarkPrice.String(), p.AverageOpenPrice.String()),
		// Orderly reports realized PnL only per closed position (position history), not on an
		// open one; left empty rather than reported as a misleading zero.
		RealizedProfit: "",
		Notional:       mulAbs(qty, p.MarkPrice.String()),
		HedgeMode:      false,
		MarginMode:     normalizeMarginMode(p.MarginMode),
		CreateTime:     ts,
		UpdateTime:     ts,
	}
}

func toOrder(o orderRow) entity.Futures_OrdersList {
	side := strings.ToUpper(strings.TrimSpace(o.Side))
	return entity.Futures_OrdersList{
		Symbol:        o.Symbol,
		OrderID:       o.OrderID.String(),
		ClientOrderID: string(o.ClientOrderID),
		PositionID:    "",
		Side:          side,
		PositionSide:  derivePositionSide(side, o.ReduceOnly),
		PositionSize:  o.Quantity.String(),
		ExecutedSize:  executedQuantity(o),
		Price:         o.Price.String(),
		Leverage:      "",
		Type:          normalizeOrderType(o.Type),
		Status:        strings.ToUpper(strings.TrimSpace(o.Status)),
		CreateTime:    o.CreatedTime.int64(),
		UpdateTime:    o.UpdatedTime.int64(),
		MarginMode:    normalizeMarginMode(o.MarginMode),
	}
}

// toAlgoOrders flattens one untriggered algo order into open-order rows.
//
// Every row carries the ROOT algo order id, because that is the id perp-api cancels by
// (DELETE /v1/algo-orders/:id). For a single-leg TP_SL — the only kind this connector places —
// that is exactly the leg. A two-leg TP_SL placed elsewhere yields two rows sharing one id, and
// cancelling either cancels both.
func toAlgoOrders(root algoOrderRow) []entity.Futures_OrdersList {
	if root.IsTriggered {
		// A triggered algo order has become a regular order, which GET /v1/orders already lists.
		return nil
	}
	rootID := root.AlgoOrderID.String()
	base := func(n algoOrderRow) entity.Futures_OrdersList {
		side := strings.ToUpper(strings.TrimSpace(n.Side))
		if side == "" {
			side = strings.ToUpper(strings.TrimSpace(root.Side))
		}
		return entity.Futures_OrdersList{
			Symbol:        root.Symbol,
			OrderID:       rootID,
			ClientOrderID: string(root.ClientOrderID),
			Side:          side,
			PositionSize:  root.Quantity.String(),
			ExecutedSize:  "0",
			// The UI shows a trigger order at its trigger level; the limit price of a LIMIT
			// variant is not the number the user set.
			Price:      n.TriggerPrice.String(),
			Status:     strings.ToUpper(strings.TrimSpace(root.AlgoStatus)),
			CreateTime: root.CreatedTime.int64(),
			UpdateTime: root.UpdatedTime.int64(),
			MarginMode: normalizeMarginMode(root.MarginMode),
		}
	}

	switch strings.ToUpper(root.AlgoType) {
	case "TP_SL", "POSITIONAL_TP_SL":
		var out []entity.Futures_OrdersList
		for _, leg := range root.ChildOrders {
			if leg.IsTriggered {
				continue
			}
			isTakeProfit := strings.EqualFold(leg.AlgoType, "TAKE_PROFIT")
			isStopLoss := strings.EqualFold(leg.AlgoType, "STOP_LOSS")
			if !isTakeProfit && !isStopLoss {
				continue
			}
			row := base(leg)
			row.PositionSide = derivePositionSide(row.Side, true)
			row.Type = triggerOrderType(isTakeProfit, leg.Type)
			row.TpOrder = isTakeProfit
			row.SlOrder = isStopLoss
			out = append(out, row)
		}
		return out
	case "STOP":
		row := base(root)
		row.PositionSide = derivePositionSide(row.Side, root.ReduceOnly)
		row.Type = triggerOrderType(false, root.Type)
		return []entity.Futures_OrdersList{row}
	default:
		// TRAILING_STOP and BRACKET are not placed by this connector; showing them with a guessed
		// type would invite a cancel through the wrong route.
		log.Printf("sumex: open algo order %s of type %q not listed", rootID, root.AlgoType)
		return nil
	}
}

func toOrderHistory(o orderRow) entity.Futures_OrdersHistory {
	side := strings.ToUpper(strings.TrimSpace(o.Side))
	return entity.Futures_OrdersHistory{
		Symbol:         o.Symbol,
		OrderID:        o.OrderID.String(),
		ClientOrderID:  string(o.ClientOrderID),
		PositionID:     "",
		Side:           side,
		PositionSide:   derivePositionSide(side, o.ReduceOnly),
		PositionSize:   o.Quantity.String(),
		ExecutedSize:   executedQuantity(o),
		Price:          o.Price.String(),
		ExecutedPrice:  o.AverageExecutedPrice.String(),
		RealisedProfit: "",
		Fee:            o.TotalFee.String(),
		FeeAsset:       o.FeeAsset,
		Leverage:       "",
		Type:           normalizeOrderType(o.Type),
		Status:         strings.ToUpper(strings.TrimSpace(o.Status)),
		HedgeMode:      false,
		MarginMode:     normalizeMarginMode(o.MarginMode),
		CreateTime:     o.CreatedTime.int64(),
		UpdateTime:     o.UpdatedTime.int64(),
	}
}

func toPositionHistory(p positionHistoryRow) entity.Futures_PositionsHistory {
	closeTime := p.CloseTimestamp.int64()
	if closeTime == 0 {
		closeTime = p.LastUpdateTime.int64()
	}
	return entity.Futures_PositionsHistory{
		Symbol:              p.Symbol,
		PositionID:          p.PositionID.String(),
		PositionSide:        strings.ToUpper(strings.TrimSpace(p.Side)),
		PositionAmt:         p.MaxPositionQty.String(),
		ExecutedPositionAmt: p.ClosedPositionQty.String(),
		AvgPrice:            p.AvgOpenPrice.String(),
		ExecutedAvgPrice:    p.AvgClosePrice.String(),
		RealisedProfit:      p.RealizedPnl.String(),
		Fee:                 p.TradingFee.String(),
		Leverage:            p.Leverage.String(),
		Funding:             p.AccumulatedFundingFee.String(),
		MarginMode:          normalizeMarginMode(string(p.MarginMode)),
		CreateTime:          p.OpenTimestamp.int64(),
		UpdateTime:          closeTime,
	}
}

func toUserTrade(t tradeRow) entity.Futures_UserTrades {
	side := strings.ToUpper(strings.TrimSpace(t.Side))
	return entity.Futures_UserTrades{
		TradeID: t.ID.String(),
		OrderID: t.OrderID.String(),
		Symbol:  t.Symbol,
		Side:    side,
		// An Orderly fill does not say whether it opened or closed a position.
		PositionSide:    "",
		Price:           t.ExecutedPrice.String(),
		Qty:             t.ExecutedQuantity.String(),
		QuoteQty:        mulAbs(t.ExecutedQuantity.String(), t.ExecutedPrice.String()),
		Commission:      t.Fee.String(),
		CommissionAsset: t.FeeAsset,
		RealisedProfit:  t.RealizedPnl.String(),
		Buyer:           side == "BUY",
		Maker:           bool(t.IsMaker),
		Time:            t.ExecutedTimestamp.int64(),
	}
}

func sortInstruments(out []entity.Futures_InstrumentsInfo) {
	sort.Slice(out, func(i, j int) bool { return out[i].Symbol < out[j].Symbol })
}

var errNoSymbol = errors.New("sumex: symbol is required")

func formatInt(v int64) string { return strconv.FormatInt(v, 10) }
