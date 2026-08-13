package katana

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

// This file is the anti-corruption layer between Katana Perps' wire format and the shapes the rest
// of this library already speaks. Two families of types live here:
//
//   - katana* / Market: unmarshal targets whose JSON tags mirror the Katana REST API exactly
//     (API_NOTES.md section A). Field names must match the wire, not our conventions.
//   - the entity.* types every other exchange in this library also produces. The <scope>_converts
//     methods at the bottom are the only way a wire type becomes one.

// Market is both the GET /v1/markets response shape and the internal representation passed around
// the connector. Symbol carries json:"market" because that is the wire field's name.
type Market struct {
	Symbol                           string `json:"market"`
	Type                             string `json:"type"`
	Status                           string `json:"status"`
	BaseAsset                        string `json:"baseAsset"`
	QuoteAsset                       string `json:"quoteAsset"`
	StepSize                         string `json:"stepSize"`
	TickSize                         string `json:"tickSize"`
	MinimumPositionSize              string `json:"minimumPositionSize"`
	MaximumPositionSize              string `json:"maximumPositionSize"`
	InitialMarginFraction            string `json:"initialMarginFraction"`
	MaintenanceMarginFraction        string `json:"maintenanceMarginFraction"`
	BasePositionSize                 string `json:"basePositionSize"`
	IncrementalPositionSize          string `json:"incrementalPositionSize"`
	IncrementalInitialMarginFraction string `json:"incrementalInitialMarginFraction"`
	IndexPrice                       string `json:"indexPrice"`
	MakerFeeRate                     string `json:"makerFeeRate"`
	TakerFeeRate                     string `json:"takerFeeRate"`
}

// katanaPosition is the GET /v1/positions response object (API_NOTES.md section A). quantity is
// signed: negative for short positions, per the docs' note reproduced there.
type katanaPosition struct {
	Market            string `json:"market"`
	Quantity          string `json:"quantity"`
	MaximumQuantity   string `json:"maximumQuantity,omitempty"`
	EntryPrice        string `json:"entryPrice"`
	ExitPrice         string `json:"exitPrice,omitempty"`
	MarkPrice         string `json:"markPrice"`
	IndexPrice        string `json:"indexPrice,omitempty"`
	LiquidationPrice  string `json:"liquidationPrice,omitempty"`
	Value             string `json:"value,omitempty"`
	RealizedPnL       string `json:"realizedPnL"`
	UnrealizedPnL     string `json:"unrealizedPnL"`
	MarginRequirement string `json:"marginRequirement,omitempty"`
	// Deliberately unread: this wire field carries a margin fraction ("0.10828271"), not the
	// multiplier "leverage" means everywhere else in this library. toPosition derives Leverage
	// from the market's InitialMarginFraction instead.
	Leverage     string `json:"leverage,omitempty"`
	TotalFunding string `json:"totalFunding,omitempty"`
	TotalOpen    string `json:"totalOpen,omitempty"`
	TotalClose   string `json:"totalClose,omitempty"`
	// Deliberately unread, and deliberately typed loosely. API_NOTES.md documents this as a quoted
	// string ("1"), but production sends a bare number (1) and a string target fails the whole
	// positions decode with "cannot unmarshal number into Go struct field". Nothing here reads the
	// value, so RawMessage accepts either shape and keeps one wrong field from emptying the panel.
	AdlQuintile    json.RawMessage `json:"adlQuintile,omitempty"`
	OpenedByFillId string          `json:"openedByFillId,omitempty"`
	LastFillId     string          `json:"lastFillId,omitempty"`
	Time           int64           `json:"time"`
}

// katanaOrder is the "standard order response object" shared by POST /v1/orders, DELETE /v1/orders
// (partially) and GET /v1/orders (by reference — API_NOTES.md notes GET /v1/orders has no
// dedicated sample and reuses this schema). See API_NOTES.md section A, "POST /v1/orders (Create
// Order)", for the field table this mirrors.
type katanaOrder struct {
	Market                  string       `json:"market"`
	OrderId                 string       `json:"orderId"`
	ClientOrderId           string       `json:"clientOrderId,omitempty"`
	Wallet                  string       `json:"wallet,omitempty"`
	Time                    int64        `json:"time"`
	Status                  string       `json:"status"`
	ErrorCode               string       `json:"errorCode,omitempty"`
	ErrorMessage            string       `json:"errorMessage,omitempty"`
	Type                    string       `json:"type"`
	Side                    string       `json:"side"`
	OriginalQuantity        string       `json:"originalQuantity"`
	ExecutedQuantity        string       `json:"executedQuantity"`
	CumulativeQuoteQuantity string       `json:"cumulativeQuoteQuantity,omitempty"`
	AvgExecutionPrice       string       `json:"avgExecutionPrice,omitempty"`
	Price                   string       `json:"price,omitempty"`
	TriggerPrice            string       `json:"triggerPrice,omitempty"`
	TriggerType             string       `json:"triggerType,omitempty"`
	ReduceOnly              bool         `json:"reduceOnly"`
	TimeInForce             string       `json:"timeInForce,omitempty"`
	SelfTradePrevention     string       `json:"selfTradePrevention,omitempty"`
	DelegatedKey            string       `json:"delegatedKey,omitempty"`
	Fills                   []katanaFill `json:"fills,omitempty"`
}

// katanaFill is the GET /v1/fills response object, also embedded as Order.fills. realizedPnL,
// builderFee and clientOrderId are genuinely optional per API_NOTES.md; makerSide/sequence/orderId/
// fee/liquidity are additionally omitted for liquidation actions.
type katanaFill struct {
	FillId        string `json:"fillId"`
	Price         string `json:"price"`
	Quantity      string `json:"quantity"`
	QuoteQuantity string `json:"quoteQuantity"`
	Time          int64  `json:"time"`
	MakerSide     string `json:"makerSide,omitempty"`
	Sequence      int64  `json:"sequence,omitempty"`
	Market        string `json:"market,omitempty"`
	OrderId       string `json:"orderId,omitempty"`
	Side          string `json:"side,omitempty"`
	Fee           string `json:"fee,omitempty"`
	Liquidity     string `json:"liquidity,omitempty"`
	Action        string `json:"action,omitempty"`
	Position      string `json:"position,omitempty"`
	Type          string `json:"type,omitempty"`
	TxId          string `json:"txId,omitempty"`
	TxStatus      string `json:"txStatus,omitempty"`
	RealizedPnL   string `json:"realizedPnL,omitempty"`
	BuilderFee    string `json:"builderFee,omitempty"`
	ClientOrderId string `json:"clientOrderId,omitempty"`
}

// katanaCancelResult is the DELETE /v1/orders per-order response object (API_NOTES.md section A).
// orderId is absent when a `notFound` order was specified by clientOrderId.
type katanaCancelResult struct {
	OrderId       string `json:"orderId,omitempty"`
	ClientOrderId string `json:"clientOrderId,omitempty"`
	Status        string `json:"status"`
}

// katanaWallet is the GET /v1/wallets response object (API_NOTES.md section A). EquityUSD holds
// "equity", which nets in unrealized PnL.
//
// Quantity holds the wire's "quoteBalance", which is a signed quote leg rather than the collateral
// held: behind a leveraged long it goes far below zero. toBalance deliberately does not report it —
// see the note there before wiring it into anything user-facing.
type katanaWallet struct {
	Wallet              string           `json:"wallet"`
	EquityUSD           string           `json:"equity"`
	FreeCollateral      string           `json:"freeCollateral"`
	HeldCollateral      string           `json:"heldCollateral,omitempty"`
	AvailableCollateral string           `json:"availableCollateral,omitempty"`
	BuyingPower         string           `json:"buyingPower,omitempty"`
	Leverage            string           `json:"leverage,omitempty"`
	MarginRatio         string           `json:"marginRatio,omitempty"`
	Quantity            string           `json:"quoteBalance"`
	UnrealizedPnL       string           `json:"unrealizedPnL"`
	MakerFeeRate        string           `json:"makerFeeRate,omitempty"`
	TakerFeeRate        string           `json:"takerFeeRate,omitempty"`
	Positions           []katanaPosition `json:"positions,omitempty"`
}

// katanaWsTokenResponse is the GET /v1/wsToken response object (API_NOTES.md section C).
type katanaWsTokenResponse struct {
	Token string `json:"token"`
}

// ===============MARKET AND WALLET FILTERS=================

// The docs never enumerate the non-active `status` values, so filterTradableMarkets allowlists
// "active" rather than blocklisting alternatives it cannot name.
const marketStatusActive = "active"

func filterTradableMarkets(raw []Market) map[string]Market {
	out := make(map[string]Market, len(raw))
	for _, m := range raw {
		if m.Type != "perpetual" {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(m.Status), marketStatusActive) {
			continue
		}
		out[m.Symbol] = m
	}
	return out
}

// isFundedWallet reports whether a wallet holds a positive vbUSDC quoteBalance. An unparseable
// balance counts as not-funded: excluding one bad-data wallet beats failing the whole account.
func isFundedWallet(w katanaWallet) bool {
	qty, err := parseBigFloat(w.Quantity)
	if err != nil {
		return false
	}
	return qty.Sign() > 0
}

// effectiveMarket returns the catalog entry with its InitialMarginFraction replaced by the
// wallet's override when one is set. ok is false for a symbol markets() does not know — a delisted
// or halted market — and callers take the zero Market rather than aborting the whole response over
// one unresolvable row.
func effectiveMarket(mkts map[string]Market, overrides map[string]string, symbol string) (Market, bool) {
	m, ok := mkts[symbol]
	if !ok {
		return Market{}, false
	}
	if ov, hasOverride := overrides[symbol]; hasOverride && strings.TrimSpace(ov) != "" {
		m.InitialMarginFraction = ov
	}
	return m, true
}

// unknownMarketTally aggregates rows whose market markets() does not know into ONE log line: a
// page of positions or fills on a delisted market would otherwise emit one line per row.
type unknownMarketTally struct {
	count  int
	sample string
}

func (t *unknownMarketTally) add(symbol string) {
	t.count++
	if t.sample == "" {
		t.sample = symbol
	}
}

func (t *unknownMarketTally) log(noun string) {
	if t.count == 0 {
		return
	}
	log.Printf("katana: %d %s reference a market not present in markets() (e.g. %q); leverage left empty for those rows", t.count, noun, t.sample)
}

// ===============DECIMALS=================

// bigPrec is the working precision (bits) for every big.Float computation here. Money never
// round-trips through float64; 256 bits leaves enough headroom that rounding happens exactly once,
// at the final 8-decimal formatting step.
const bigPrec = 256

// parseBigFloat parses a decimal string at bigPrec precision, base 10 explicitly: the default base
// 0 would also accept hex floats ("0x1p4") and underscore grouping ("1_0"), which Katana never
// sends. Base 10 still parses the literal "Inf", so IsInf is checked separately.
func parseBigFloat(s string) (*big.Float, error) {
	f := new(big.Float).SetPrec(bigPrec)
	parsed, _, err := f.Parse(strings.TrimSpace(s), 10)
	if err != nil {
		return nil, fmt.Errorf("katana: %q is not a valid decimal number: %w", s, err)
	}
	if parsed.IsInf() {
		return nil, fmt.Errorf("katana: %q is not a finite decimal number", s)
	}
	return parsed, nil
}

func parsePositiveDecimal(s, label string) (*big.Float, error) {
	f, err := parseBigFloat(s)
	if err != nil {
		return nil, fmt.Errorf("katana: %s %w", label, err)
	}
	if f.Sign() <= 0 {
		return nil, fmt.Errorf("katana: %s %q must be a positive number", label, s)
	}
	return f, nil
}

// formatDecimal8 renders f as a zero-padded, fixed 8-decimal string ("0.10000000"), per the global
// "money is strings, 8 decimals" contract.
func formatDecimal8(f *big.Float) string {
	return f.Text('f', 8)
}

// trimTrailingZeros strips a redundant fractional part ("20.00000000" -> "20"). Leverage is
// rendered as a bare number, unlike prices and quantities.
func trimTrailingZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// leverageToIMF converts a leverage multiplier ("10") to the initial margin fraction Katana
// operates on ("0.10000000"): IMF = 1 / leverage.
func leverageToIMF(leverage string) (string, error) {
	lev, err := parsePositiveDecimal(leverage, "leverage")
	if err != nil {
		return "", err
	}
	imf := new(big.Float).SetPrec(bigPrec).Quo(big.NewFloat(1), lev)
	return formatDecimal8(imf), nil
}

// plainDecimalPattern gates parseDecimalRat: big.Rat.SetString on its own also accepts quotients
// ("3/2"), exponents and base prefixes, none of which Katana ever sends.
var plainDecimalPattern = regexp.MustCompile(`^[+-]?(\d+(\.\d*)?|\.\d+)$`)

// parseDecimalRat parses a decimal EXACTLY, as a rational rather than a binary float. The tiered-
// margin calculation divides one wire decimal by another and then takes a ceiling: a binary-float
// 0.3/0.1 can land a hair above 3 and ceil to 4 — a whole extra margin tier — at any precision.
// big.Rat has no rounding error to land on the wrong side of.
func parseDecimalRat(s string) (*big.Rat, error) {
	t := strings.TrimSpace(s)
	if !plainDecimalPattern.MatchString(t) {
		return nil, fmt.Errorf("katana: %q is not a valid decimal number", s)
	}
	r, ok := new(big.Rat).SetString(t)
	if !ok {
		return nil, fmt.Errorf("katana: %q is not a valid decimal number", s)
	}
	return r, nil
}

// ceilPositiveRat returns the smallest integer >= r, for r >= 0. Num/Denom are always exact, so
// QuoRem's truncation toward zero is a floor here and the remainder test is exact — no epsilon, no
// float64.
func ceilPositiveRat(r *big.Rat) *big.Int {
	q, rem := new(big.Int).QuoRem(r.Num(), r.Denom(), new(big.Int))
	if rem.Sign() != 0 {
		q.Add(q, big.NewInt(1))
	}
	return q
}

// effectiveIMFForSize returns the initial margin fraction that actually applies to a position of
// |quantity| in market m. Katana's margin schedule is tiered — per API_NOTES.md, "each step of
// `incrementalPositionSize` increases the `initialMarginFraction` by
// `incrementalInitialMarginFraction`":
//
//	|q| <= basePositionSize  ->  imf = initialMarginFraction
//	|q| >  basePositionSize  ->  imf = initialMarginFraction +
//	                                   ceil((|q| - basePositionSize) / incrementalPositionSize)
//	                                   * incrementalInitialMarginFraction
//
// A partial step counts as a full one. The docs do not say which way to round; ceil is the
// conservative direction because a higher IMF reports a LOWER leverage, and understating leverage
// is the safe error here. NEEDS LIVE-SANDBOX CONFIRMATION.
//
// A market with no usable tier schedule returns its own initialMarginFraction unchanged, but an
// unparseable quantity or initialMarginFraction is an error — the caller renders that as an
// unknown leverage rather than a confident number that may be wrong.
//
// Overrides: callers pass m through effectiveMarket, which may already have replaced
// InitialMarginFraction with the wallet's override, so an override becomes the base tier here.
// Katana's engine may instead take max(override, tieredFromMarketDefault); the docs say nothing.
// The two agree whenever no override is set, and this one reports the lower leverage otherwise.
// NEEDS LIVE-SANDBOX CONFIRMATION.
func effectiveIMFForSize(m Market, quantity string) (string, error) {
	baseIMF, err := parseDecimalRat(m.InitialMarginFraction)
	if err != nil {
		return "", fmt.Errorf("katana: initialMarginFraction %w", err)
	}
	if baseIMF.Sign() <= 0 {
		return "", fmt.Errorf("katana: initialMarginFraction %q must be a positive number", m.InitialMarginFraction)
	}

	basePositionSize, errBase := parseDecimalRat(m.BasePositionSize)
	incrementalPositionSize, errIncrSize := parseDecimalRat(m.IncrementalPositionSize)
	incrementalIMF, errIncrIMF := parseDecimalRat(m.IncrementalInitialMarginFraction)
	if errBase != nil || errIncrSize != nil || errIncrIMF != nil {
		return m.InitialMarginFraction, nil
	}
	if basePositionSize.Sign() < 0 || incrementalPositionSize.Sign() <= 0 || incrementalIMF.Sign() < 0 {
		return m.InitialMarginFraction, nil
	}

	qty, err := parseDecimalRat(quantity)
	if err != nil {
		return "", fmt.Errorf("katana: position quantity %w", err)
	}
	qty.Abs(qty)
	if qty.Cmp(basePositionSize) <= 0 {
		return m.InitialMarginFraction, nil
	}

	excess := new(big.Rat).Sub(qty, basePositionSize)
	steps := ceilPositiveRat(new(big.Rat).Quo(excess, incrementalPositionSize))
	tiered := new(big.Rat).Add(baseIMF, new(big.Rat).Mul(incrementalIMF, new(big.Rat).SetInt(steps)))
	return tiered.FloatString(8), nil
}

// imfToLeverage converts an initial margin fraction ("0.05000000") back to the leverage multiplier
// the rest of the stack understands ("20"): leverage = 1 / IMF.
func imfToLeverage(imf string) (string, error) {
	frac, err := parsePositiveDecimal(imf, "initialMarginFraction")
	if err != nil {
		return "", err
	}
	lev := new(big.Float).SetPrec(bigPrec).Quo(big.NewFloat(1), frac)
	return trimTrailingZeros(formatDecimal8(lev)), nil
}

// absDecimal8 returns |s| formatted to 8 zero-padded decimals, falling back to a plain "-" strip on
// unparseable input because its caller (toPosition) has no error channel.
func absDecimal8(s string) string {
	f, err := parseBigFloat(s)
	if err != nil {
		return strings.TrimPrefix(strings.TrimSpace(s), "-")
	}
	f.Abs(f)
	return formatDecimal8(f)
}

// isNegativeDecimal reports whether a decimal string is negative. Same fallback as absDecimal8.
func isNegativeDecimal(s string) bool {
	f, err := parseBigFloat(s)
	if err != nil {
		return strings.HasPrefix(strings.TrimSpace(s), "-")
	}
	return f.Sign() < 0
}

// isZeroDecimal reports whether a decimal string parses to exactly zero. An unparseable value reads
// as not-zero, so one bad data point cannot silently drop a real row from a response.
func isZeroDecimal(s string) bool {
	f, err := parseBigFloat(s)
	if err != nil {
		return false
	}
	return f.Sign() == 0
}

// mulAbsDecimal8 returns |quantity| * price for a position's notional. Unparseable input yields ""
// (unknown), never "0.00000000": a money field reading zero is indistinguishable from a genuinely
// zero notional, and unlike absDecimal8 there is no partially-correct product to degrade to.
func mulAbsDecimal8(quantity, price string) string {
	qf, errQ := parseBigFloat(quantity)
	pf, errP := parseBigFloat(price)
	if errQ != nil || errP != nil {
		return ""
	}
	qf.Abs(qf)
	product := new(big.Float).SetPrec(bigPrec).Mul(qf, pf)
	return formatDecimal8(product)
}

// subDecimal8 returns a - b formatted to 8 zero-padded decimals. Unparseable input degrades to a
// trimmed `a` rather than "" or "0.00000000": the only caller is the balance row, where an empty
// string renders as no account and a zero renders as an empty one, while `a` (equity) is the
// closest honest number available.
func subDecimal8(a, b string) string {
	af, errA := parseBigFloat(a)
	bf, errB := parseBigFloat(b)
	if errA != nil || errB != nil {
		return strings.TrimSpace(a)
	}
	return formatDecimal8(new(big.Float).SetPrec(bigPrec).Sub(af, bf))
}

// decimalPlaces counts significant fractional digits ("0.00000100" -> "6"), deriving size/
// pricePrecision from stepSize/tickSize. Katana zero-pads to 8 decimals, so trailing zeros are not
// significant.
func decimalPlaces(s string) string {
	s = strings.TrimSpace(s)
	dot := strings.Index(s, ".")
	if dot < 0 {
		return "0"
	}
	frac := strings.TrimRight(s[dot+1:], "0")
	return strconv.Itoa(len(frac))
}

// ===============ROW MAPPERS=================

// toPosition maps a katanaPosition to entity.Futures_Positions. m must already carry the wallet's
// effective (override-aware) initial margin fraction; toPosition does no lookup itself. Leverage
// uses the TIERED fraction the position's own size attracts — this is the only place the size is
// known, so the only place the tier schedule can apply. Either step failing leaves Leverage empty
// rather than reporting a number that may be overstated.
func toPosition(p katanaPosition, m Market) entity.Futures_Positions {
	side := "LONG"
	if isNegativeDecimal(p.Quantity) {
		side = "SHORT"
	}

	leverage := ""
	if imf, err := effectiveIMFForSize(m, p.Quantity); err == nil {
		if lev, lerr := imfToLeverage(imf); lerr == nil {
			leverage = lev
		}
	}

	return entity.Futures_Positions{
		Symbol:       p.Market,
		PositionSide: side,
		PositionSize: absDecimal8(p.Quantity),
		Leverage:     leverage,
		// Katana nets one signed position per market and has no native position id; left empty
		// rather than guessed, as the hyperliquid connector does for the same shape.
		PositionID:       "",
		EntryPrice:       p.EntryPrice,
		MarkPrice:        p.MarkPrice,
		UnRealizedProfit: p.UnrealizedPnL,
		RealizedProfit:   p.RealizedPnL,
		Notional:         mulAbsDecimal8(p.Quantity, p.MarkPrice),
		HedgeMode:        false,
		MarginMode:       "CROSS",
		CreateTime:       p.Time,
		UpdateTime:       p.Time,
	}
}

// classifyTriggerOrder reports whether a Katana order type string (API_NOTES.md's "Order Type
// Values": market, limit, stopLossMarket, stopLossLimit, takeProfitMarket, takeProfitLimit) is a
// take-profit or stop-loss order.
func classifyTriggerOrder(orderType string) (isTakeProfit, isStopLoss bool) {
	switch strings.ToLower(strings.TrimSpace(orderType)) {
	case "takeprofitmarket", "takeprofitlimit":
		return true, false
	case "stoplossmarket", "stoplosslimit":
		return false, true
	default:
		return false, false
	}
}

// derivePositionSide infers positionSide from an order's own side. A closing order (explicitly
// reduce-only, or a TP/SL, which this connector always places reduce-only) targets the OPPOSITE
// side from an opening order of the same side — without the inversion a reduce-only sell that
// closes a long renders in the UI as "Open Short".
func derivePositionSide(side string, reduceOnly, isTakeProfit, isStopLoss bool) string {
	closing := reduceOnly || isTakeProfit || isStopLoss
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

// resolveOrderPrice prefers triggerPrice for trigger orders, as the hyperliquid connector does.
// Two wire quirks make it necessary: Katana omits `price` entirely on *Market trigger variants (the
// frontend's `price ?? fallback` does not treat "" as nullish, so a live stop-loss would render at
// 0), and on *Limit variants `price` is the limit price, not the level the UI must show.
func resolveOrderPrice(o katanaOrder) string {
	isTakeProfit, isStopLoss := classifyTriggerOrder(o.Type)
	triggerPrice := strings.TrimSpace(o.TriggerPrice)
	if (isTakeProfit || isStopLoss) && triggerPrice != "" {
		return triggerPrice
	}
	return o.Price
}

// orderDerivedFields bundles the values toOrder and toOrderHistory both compute from a katanaOrder's
// side/type/reduceOnly, so the two mappers can't drift apart on them.
type orderDerivedFields struct {
	side         string
	positionSide string
	price        string
	isTakeProfit bool
	isStopLoss   bool
}

func deriveOrderFields(o katanaOrder) orderDerivedFields {
	side := strings.ToUpper(strings.TrimSpace(o.Side))
	isTakeProfit, isStopLoss := classifyTriggerOrder(o.Type)
	return orderDerivedFields{
		side:         side,
		positionSide: derivePositionSide(side, o.ReduceOnly, isTakeProfit, isStopLoss),
		price:        resolveOrderPrice(o),
		isTakeProfit: isTakeProfit,
		isStopLoss:   isStopLoss,
	}
}

// normalizeOrderType maps Katana's order type onto the platform vocabulary the API validates
// against. Uppercasing the wire value is not enough: "takeProfitMarket" becomes
// "TAKEPROFITMARKET", which is not a member of the shared OrderType enum, so a single trigger
// order made the whole open-orders (and orders-history) response fail output validation as
// "Invalid response format" — an empty panel caused by one resting TP/SL.
//
// The six values below are the complete REST-layer set per API_NOTES.md ("Order Type Values").
// The targets follow the Binance/BingX naming this enum was modelled on, where the limit variant
// is STOP / TAKE_PROFIT and the market variant is STOP_MARKET / TAKE_PROFIT_MARKET.
//
// Anything unrecognised degrades to MARKET or LIMIT by suffix rather than passing an unmappable
// value through: a coarse-but-valid type costs one row its exact label, whereas an invalid one
// costs the caller every row in the response.
func normalizeOrderType(wireType string) string {
	switch strings.ToLower(strings.TrimSpace(wireType)) {
	case "market":
		return "MARKET"
	case "limit":
		return "LIMIT"
	case "stoplossmarket":
		return "STOP_MARKET"
	case "stoplosslimit":
		return "STOP"
	case "takeprofitmarket":
		return "TAKE_PROFIT_MARKET"
	case "takeprofitlimit":
		return "TAKE_PROFIT"
	}

	normalized := "LIMIT"
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(wireType)), "market") {
		normalized = "MARKET"
	}
	log.Printf("katana: unknown order type %q reported as %s", wireType, normalized)
	return normalized
}

// toOrder maps a katanaOrder to entity.Futures_OrdersList. Katana orders carry no position context
// (no hedge mode, one netted position per market), so PositionSize reuses the order's own quantity
// and PositionID/Leverage stay empty, matching the hyperliquid precedent. MarginMode is the one
// deviation from it: on Katana "CROSS" is an invariant, so it is known.
//
// The wire's reduceOnly flag has nowhere to land: entity.Futures_OrdersList has no such field. It
// is still read here, by deriveOrderFields, to invert positionSide for a closing order.
func toOrder(o katanaOrder) entity.Futures_OrdersList {
	d := deriveOrderFields(o)

	return entity.Futures_OrdersList{
		Symbol:        o.Market,
		OrderID:       o.OrderId,
		ClientOrderID: o.ClientOrderId,
		PositionID:    "",
		Side:          d.side,
		PositionSide:  d.positionSide,
		PositionSize:  o.OriginalQuantity,
		ExecutedSize:  o.ExecutedQuantity,
		Price:         d.price,
		Leverage:      "",
		Type:          normalizeOrderType(o.Type),
		Status:        strings.ToUpper(strings.TrimSpace(o.Status)),
		CreateTime:    o.Time,
		UpdateTime:    o.Time,
		MarginMode:    "CROSS",
		TpOrder:       d.isTakeProfit,
		SlOrder:       d.isStopLoss,
	}
}

// toOrderHistory maps a closed katanaOrder to entity.Futures_OrdersHistory, sharing
// deriveOrderFields with toOrder so a positionSide or price fix cannot land in one and miss the
// other. RealisedProfit/Fee/FeeAsset need cross-fill aggregation and Leverage needs a per-market
// lookup, neither of which this signature can reach; the orders-history action fills all four in.
func toOrderHistory(o katanaOrder) entity.Futures_OrdersHistory {
	d := deriveOrderFields(o)

	return entity.Futures_OrdersHistory{
		Symbol:         o.Market,
		OrderID:        o.OrderId,
		ClientOrderID:  o.ClientOrderId,
		PositionID:     "",
		Side:           d.side,
		PositionSide:   d.positionSide,
		PositionSize:   o.OriginalQuantity,
		ExecutedSize:   o.ExecutedQuantity,
		Price:          d.price,
		ExecutedPrice:  o.AvgExecutionPrice,
		RealisedProfit: "",
		Fee:            "",
		FeeAsset:       "",
		Leverage:       "",
		Type:           normalizeOrderType(o.Type),
		Status:         strings.ToUpper(strings.TrimSpace(o.Status)),
		HedgeMode:      false,
		MarginMode:     "CROSS",
		CreateTime:     o.Time,
		UpdateTime:     o.Time,
		TpOrder:        d.isTakeProfit,
		SlOrder:        d.isStopLoss,
	}
}

// toBalance maps a katanaWallet to the single-entry slice the contract expects. Katana's only
// collateral is vbUSDC, reported as "USDC" because dashboard, holdings and pricing key on that.
//
// Balance is NOT the wire's quoteBalance. That field is a signed quote leg: opening a leveraged
// long drives it far below zero (3.63 USDC of collateral behind a 53.78 position reads about -50),
// which is not a wallet balance in the sense every other exchange here means. Deriving it as
// equity - unrealizedPnL instead matches the rest of the library, where Balance excludes unrealized
// PnL and Equity includes it, and is non-negative by construction: equity IS collateral plus
// unrealized PnL, so the difference is the collateral itself.
func toBalance(w katanaWallet) []entity.FuturesBalance {
	return []entity.FuturesBalance{
		{
			Asset:            "USDC",
			Balance:          subDecimal8(w.EquityUSD, w.UnrealizedPnL),
			Equity:           w.EquityUSD,
			Available:        w.FreeCollateral,
			UnrealizedProfit: w.UnrealizedPnL,
		},
	}
}

// toInstrumentInfo maps a Market to entity.Futures_InstrumentsInfo. contractSize "1"/isSizeContract
// false because Katana has no contract-multiplier markets. State is the platform-normalised "LIVE",
// not Katana's own "online": consumers gate on `state === "LIVE"`, so emitting "online" filters
// every Katana pair out.
//
// MaxLeverage is deliberately the BASE tier's leverage — that genuinely is the maximum a market
// offers, which is what the field means. The consequence is that a sizing slider can still see an
// order rejected above basePositionSize; effectiveIMFForSize applies the tiers where the position
// size is actually known.
func toInstrumentInfo(m Market) entity.Futures_InstrumentsInfo {
	maxLeverage, err := imfToLeverage(m.InitialMarginFraction)
	if err != nil {
		maxLeverage = ""
	}
	return entity.Futures_InstrumentsInfo{
		Symbol:         m.Symbol,
		Base:           m.BaseAsset,
		Quote:          m.QuoteAsset,
		MinQty:         m.MinimumPositionSize,
		PricePrecision: decimalPlaces(m.TickSize),
		SizePrecision:  decimalPlaces(m.StepSize),
		State:          "LIVE",
		MaxLeverage:    maxLeverage,
		ContractSize:   "1",
		IsSizeContract: false,
	}
}

// toPositionHistory maps a closing katanaFill to entity.Futures_PositionsHistory. PositionID is
// empty for the same reason as in toPosition. Funding is empty because a fill carries none — it
// lives on GET /v1/fundingPayments, which this connector does not call.
func toPositionHistory(f katanaFill, leverage string) entity.Futures_PositionsHistory {
	return entity.Futures_PositionsHistory{
		Symbol:              f.Market,
		PositionID:          "",
		PositionSide:        strings.ToUpper(strings.TrimSpace(f.Position)),
		PositionAmt:         f.Quantity,
		ExecutedPositionAmt: f.Quantity,
		AvgPrice:            f.Price,
		ExecutedAvgPrice:    f.Price,
		RealisedProfit:      f.RealizedPnL,
		Leverage:            leverage,
		Fee:                 f.Fee,
		Funding:             "",
		MarginMode:          "CROSS",
		CreateTime:          f.Time,
		UpdateTime:          f.Time,
	}
}

// toUserTrade maps a katanaFill to entity.Futures_UserTrades. PositionSide comes straight off the
// fill's own `position` field rather than being derived — Katana states it.
func toUserTrade(f katanaFill) entity.Futures_UserTrades {
	side := strings.ToUpper(strings.TrimSpace(f.Side))
	return entity.Futures_UserTrades{
		TradeID:         f.FillId,
		OrderID:         f.OrderId,
		Symbol:          f.Market,
		Side:            side,
		PositionSide:    strings.ToUpper(strings.TrimSpace(f.Position)),
		Price:           f.Price,
		Qty:             f.Quantity,
		QuoteQty:        f.QuoteQuantity,
		Commission:      f.Fee,
		CommissionAsset: "USDC",
		RealisedProfit:  f.RealizedPnL,
		Buyer:           side == "BUY",
		Maker:           strings.EqualFold(strings.TrimSpace(f.Liquidity), "maker"),
		BuilderFee:      strings.TrimSpace(f.BuilderFee),
		Time:            f.Time,
	}
}

// ===============CONVERTS=================

type account_converts struct{}

// convertAccountInfo reports what a Katana API account can do. Katana exposes no scope
// introspection — the docs state what scope each endpoint REQUIRES, never what the calling key was
// granted — and a read-only action must not attempt a trade to find out. CanTrade true is the
// least-bad guess: it mislabels a read-only key as TRADE (the attempt then fails with Katana's own
// scope error) rather than hiding working functionality on a full-scope key.
func (c *account_converts) convertAccountInfo(wallet string) (out entity.AccountInformation) {
	out.UID = wallet
	out.Label = ""
	out.IP = "0.0.0.0/0"
	out.CanRead = true
	out.CanTrade = true
	out.CanTransfer = false
	// Perpetuals only — Katana lists no spot markets at all.
	out.PermSpot = false
	out.PermFutures = true
	return out
}

// convertSignAuthStream carries Katana's private-channel WebSocket token in the Signature slot.
// Katana has no WebSocket signature — its opaque server-issued token rides there instead, because
// consumers read `response.signature` whatever kind of value it carries.
func (c *account_converts) convertSignAuthStream(in katanaWsTokenResponse) (out entity.SignAuthStream) {
	out.Signature = in.Token
	return out
}

type futures_converts struct{}

func (c *futures_converts) convertInstrumentsInfo(in map[string]Market) (out []entity.Futures_InstrumentsInfo) {
	symbols := make([]string, 0, len(in))
	for sym := range in {
		symbols = append(symbols, sym)
	}
	sort.Strings(symbols)

	out = make([]entity.Futures_InstrumentsInfo, 0, len(in))
	for _, sym := range symbols {
		out = append(out, toInstrumentInfo(in[sym]))
	}
	return out
}

func (c *futures_converts) convertBalance(in katanaWallet) (out []entity.FuturesBalance) {
	return toBalance(in)
}

// convertPositions drops zero-quantity rows: Katana returns closed positions with quantity "0",
// which would otherwise render as a phantom size-0 LONG. overrides may be nil — it only decorates a
// leverage number and must never empty the whole positions panel.
func (c *futures_converts) convertPositions(in []katanaPosition, mkts map[string]Market, overrides map[string]string) (out []entity.Futures_Positions) {
	var unknown unknownMarketTally
	out = make([]entity.Futures_Positions, 0, len(in))
	for _, p := range in {
		if isZeroDecimal(p.Quantity) {
			continue
		}
		m, ok := effectiveMarket(mkts, overrides, p.Market)
		if !ok {
			unknown.add(p.Market)
		}
		out = append(out, toPosition(p, m))
	}
	unknown.log("position(s)")
	return out
}

func (c *futures_converts) convertOrderList(in []katanaOrder) (out []entity.Futures_OrdersList) {
	out = make([]entity.Futures_OrdersList, 0, len(in))
	for _, o := range in {
		out = append(out, toOrder(o))
	}
	return out
}

func (c *futures_converts) convertOrdersHistory(in []katanaOrder) (out []entity.Futures_OrdersHistory) {
	out = make([]entity.Futures_OrdersHistory, 0, len(in))
	for _, o := range in {
		out = append(out, toOrderHistory(o))
	}
	return out
}

// convertPositionsHistory resolves each row's leverage from the fill's OWN market, mirroring
// convertPositions: a page of closing fills can span several markets, and one leverage applied to
// all of them would report another market's number. Plain imfToLeverage, not effectiveIMFForSize —
// a fill's quantity is what closed, not the position size the tier schedule keys on. overrides may
// be nil. An unknown market leaves that row's leverage empty rather than failing the response.
func (c *futures_converts) convertPositionsHistory(in []katanaFill, mkts map[string]Market, overrides map[string]string) (out []entity.Futures_PositionsHistory) {
	var unknown unknownMarketTally
	out = make([]entity.Futures_PositionsHistory, 0, len(in))
	for _, f := range in {
		leverage := ""
		m, ok := effectiveMarket(mkts, overrides, f.Market)
		if ok {
			if lev, lerr := imfToLeverage(m.InitialMarginFraction); lerr == nil {
				leverage = lev
			}
		} else {
			unknown.add(f.Market)
		}
		out = append(out, toPositionHistory(f, leverage))
	}
	unknown.log("fill(s)")
	return out
}

func (c *futures_converts) convertUserTrades(in []katanaFill) (out []entity.Futures_UserTrades) {
	out = make([]entity.Futures_UserTrades, 0, len(in))
	for _, f := range in {
		out = append(out, toUserTrade(f))
	}
	return out
}

// convertLeverage reports one symbol's effective leverage. Long and short carry the same value:
// Katana is cross-margin, one-way, so there is only ever one margin requirement per market.
func (c *futures_converts) convertLeverage(symbol, leverage string) (out entity.Futures_Leverage) {
	out.Symbol = symbol
	out.Leverage = leverage
	out.MarginMode = "CROSS"
	return out
}

// convertPositionMode answers from an invariant: Katana has no hedge mode, always one-way.
func (c *futures_converts) convertPositionMode() (out entity.Futures_PositionsMode) {
	out.HedgeMode = false
	return out
}

// convertMarginMode answers from an invariant: Katana is cross-margin only.
func (c *futures_converts) convertMarginMode() (out entity.Futures_MarginMode) {
	out.MarginMode = "CROSS"
	return out
}

// convertPlaceOrder maps the standard order response POST /v1/orders returns to the single-entry
// slice the place/amend contract expects. PositionID stays empty: Katana nets one position per
// market and has no position id to report, exactly as toPosition explains.
func (c *futures_converts) convertPlaceOrder(in katanaOrder) (out []entity.PlaceOrder) {
	return []entity.PlaceOrder{
		{
			OrderID:       in.OrderId,
			ClientOrderID: in.ClientOrderId,
			Ts:            in.Time,
		},
	}
}

// convertCancelOrder maps DELETE /v1/orders results. ts is client-observed and supplied by the
// caller: the response carries only orderId/clientOrderId/status, so no exchange timestamp exists.
// Confirming the `status` is the ACTION's job — a row reaching here is already confirmed canceled.
func (c *futures_converts) convertCancelOrder(in []katanaCancelResult, ts int64) (out []entity.PlaceOrder) {
	out = make([]entity.PlaceOrder, 0, len(in))
	for _, item := range in {
		out = append(out, entity.PlaceOrder{
			OrderID:       item.OrderId,
			ClientOrderID: item.ClientOrderId,
			Ts:            ts,
		})
	}
	return out
}
