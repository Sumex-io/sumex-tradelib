package xemus

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
		return "", fmt.Errorf("xemus: %s %q is not a plain decimal", label, s)
	}
	r, _ := parseRat(s)
	if r.Sign() <= 0 {
		return "", fmt.Errorf("xemus: %s must be greater than zero", label)
	}
	return json.Number(s), nil
}

// wireLeverage converts a leverage string to the integer perp-api takes (1..100). "10" and "10.0"
// are accepted; a fractional leverage is refused rather than silently truncated.
func wireLeverage(s string) (int64, error) {
	r, ok := parseRat(s)
	if !ok || !r.IsInt() {
		return 0, fmt.Errorf("xemus: leverage %q must be a whole number", s)
	}
	v := r.Num()
	if !v.IsInt64() || v.Int64() < 1 || v.Int64() > 100 {
		return 0, fmt.Errorf("xemus: leverage %q must be between 1 and 100", s)
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
		return "", "", fmt.Errorf("xemus: unexpected symbol %q", symbol)
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
		log.Printf("xemus: unknown order type %q reported as LIMIT", t)
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

// toBalance returns one row per collateral token, amounts in that token, following the library
// convention: Balance excludes unrealized PnL, Equity includes it.
//
// Orderly settles every position's PnL in USDC, so the USDC row carries all of it: its equity is
// the USDC holding plus the unsettled PnL of every position plus the margin set aside for
// isolated positions, the same account value Orderly's SDK shows. Its Available is the account's
// free collateral, which is in USD and already counts the other tokens at their haircut, because
// that is what an order can use. The row is returned even at a zero holding, since a user
// trading on other collateral still has PnL and free collateral there.
//
// Any other token's row is its holding. Orderly caps withdrawing it by the free collateral, which
// a per-token figure cannot express, so its Available is the holding too.
func toBalance(h holdingsResponse, p positionsResponse) []entity.FuturesBalance {
	usdc := new(big.Rat)
	var others []entity.FuturesBalance
	for _, row := range h.Holding {
		amount, ok := parseRat(row.Holding.String())
		if !ok {
			continue
		}
		if row.Token == "USDC" {
			usdc.Add(usdc, amount)
			continue
		}
		if amount.Sign() == 0 {
			continue
		}
		qty := ratString(amount)
		others = append(others, entity.FuturesBalance{Asset: row.Token, Balance: qty, Equity: qty, Available: qty, UnrealizedProfit: "0"})
	}

	equity := new(big.Rat).Set(usdc)
	unrealized := new(big.Rat)
	for _, row := range p.Rows {
		if u, ok := parseRat(row.UnsettledPnl.String()); ok {
			equity.Add(equity, u)
		}
		if row.MarginMode == "ISOLATED" {
			if m, ok := parseRat(row.Margin.String()); ok {
				equity.Add(equity, m)
			}
		}
		if u, ok := parseRat(unrealizedPnl(row.PositionQty.String(), row.MarkPrice.String(), row.AverageOpenPrice.String())); ok {
			unrealized.Add(unrealized, u)
		}
	}
	upnl := ratString(unrealized)
	return append([]entity.FuturesBalance{{
		Asset:            "USDC",
		Balance:          ratString(new(big.Rat).Sub(equity, unrealized)),
		Equity:           ratString(equity),
		Available:        p.FreeCollateral.String(),
		UnrealizedProfit: upnl,
	}}, others...)
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

// toAlgoOrders flattens one open algo order into open-order rows.
//
// TP/SL rows carry the ROOT algo order id with TpOrder/SlOrder set, because that is the id
// perp-api cancels by (DELETE /v1/algo-orders/:id) and the flag is how the platform routes the
// cancel there. For a single-leg TP_SL — the kind this connector places on an open position — that
// is exactly the leg. A two-leg TP/SL yields two rows sharing one id: cancelling either cancels
// both, while an amend names its leg through TpOrder/SlOrder.
//
// A row that is an algo order but not a TP/SL — a stop order, or a bracket whose entry is not yet
// placed — gets no flag, so its id carries algoOrderIDPrefix instead.
func toAlgoOrders(root algoOrderRow) []entity.Futures_OrdersList {
	rootID := root.AlgoOrderID.String()
	// A leg group surfaced on its own row still cancels through the tree it belongs to.
	if r := root.RootAlgoOrderID.String(); r != "" && r != "0" {
		rootID = r
	}
	base := func(n algoOrderRow, quantity num) entity.Futures_OrdersList {
		side := strings.ToUpper(strings.TrimSpace(n.Side))
		if side == "" {
			side = strings.ToUpper(strings.TrimSpace(root.Side))
		}
		return entity.Futures_OrdersList{
			Symbol:        root.Symbol,
			OrderID:       rootID,
			ClientOrderID: string(root.ClientOrderID),
			Side:          side,
			PositionSize:  quantity.String(),
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
	legRows := func(group algoOrderRow) []entity.Futures_OrdersList {
		var out []entity.Futures_OrdersList
		for _, leg := range group.ChildOrders {
			if bool(leg.IsTriggered) || !leg.isLive() {
				continue
			}
			isTakeProfit := strings.EqualFold(leg.AlgoType, "TAKE_PROFIT")
			isStopLoss := strings.EqualFold(leg.AlgoType, "STOP_LOSS")
			if !isTakeProfit && !isStopLoss {
				continue
			}
			row := base(leg, group.Quantity)
			row.PositionSide = derivePositionSide(row.Side, true)
			row.Type = triggerOrderType(isTakeProfit, leg.Type)
			row.TpOrder = isTakeProfit
			row.SlOrder = isStopLoss
			out = append(out, row)
		}
		return out
	}

	switch strings.ToUpper(root.AlgoType) {
	case "TP_SL", "POSITIONAL_TP_SL":
		if root.IsTriggered {
			return nil
		}
		return legRows(root)
	case "STOP":
		if root.IsTriggered {
			// A triggered stop has become a regular order, which GET /v1/orders already lists.
			return nil
		}
		row := base(root, root.Quantity)
		row.OrderID = algoOrderRef(rootID)
		row.PositionSide = derivePositionSide(row.Side, root.ReduceOnly)
		row.Type = triggerOrderType(false, root.Type)
		return []entity.Futures_OrdersList{row}
	case "BRACKET":
		if !root.IsTriggered {
			// The entry itself, still an algo order: shown as the MARKET / LIMIT order it is, at
			// its own price. Its TP/SL legs are not live until it fills, so they are not listed.
			row := base(root, root.Quantity)
			row.OrderID = algoOrderRef(rootID)
			row.Price = root.Price.String()
			row.PositionSide = derivePositionSide(row.Side, false)
			row.Type = normalizeOrderType(root.Type)
			return []entity.Futures_OrdersList{row}
		}
		// The entry has been placed; what remains open is the position TP/SL.
		var out []entity.Futures_OrdersList
		for _, group := range root.ChildOrders {
			if strings.EqualFold(group.AlgoType, "POSITIONAL_TP_SL") && !bool(group.IsTriggered) {
				out = append(out, legRows(group)...)
			}
		}
		return out
	default:
		// TRAILING_STOP is not placed by this connector; showing it with a guessed type would
		// invite an edit through the wrong fields.
		log.Printf("xemus: open algo order %s of type %q not listed", rootID, root.AlgoType)
		return nil
	}
}

func (a algoOrderRow) isLive() bool { return a.IsActivated == nil || bool(*a.IsActivated) }

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

var errNoSymbol = errors.New("xemus: symbol is required")

func formatInt(v int64) string { return strconv.FormatInt(v, 10) }
