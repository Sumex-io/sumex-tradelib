package katana

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_ordersHistory struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       futures_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)
	markets       func(ctx context.Context, opts ...utils.RequestOption) (map[string]Market, error)

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
}

func (s *futures_ordersHistory) Symbol(symbol string) *futures_ordersHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_ordersHistory) StartTime(startTime int64) *futures_ordersHistory {
	s.startTime = &startTime
	return s
}

func (s *futures_ordersHistory) EndTime(endTime int64) *futures_ordersHistory {
	s.endTime = &endTime
	return s
}

func (s *futures_ordersHistory) Limit(limit int64) *futures_ordersHistory {
	s.limit = &limit
	return s
}

// Do maps GET /v1/orders (closed=true), filling in the four fields convertOrdersHistory's input
// cannot reach: RealisedProfit/Fee/FeeAsset summed across the order's fills, and the override-aware
// Leverage. Whether GET /v1/orders inlines its fills is undocumented (API_NOTES.md Gap 6), so
// resolveMissingOrderFills backfills executed orders that arrive without them.
//
// Results are sorted newest-first: Katana's wire order is undocumented, but pagers page backward
// assuming the LAST row is the OLDEST, which only holds if we sort explicitly.
func (s *futures_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersHistory, err error) {
	wallet, err := s.resolveWallet(ctx, opts...)
	if err != nil {
		return res, err
	}
	mkts, err := s.markets(ctx, opts...)
	if err != nil {
		return res, err
	}

	nonce, _, err := newNonce()
	if err != nil {
		return res, err
	}

	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v1/orders",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(utils.Params{
		"nonce":  nonce,
		"wallet": wallet,
		"closed": "true",
	})
	marketFilter := applyHistoryPaging(r, s.symbol, s.startTime, s.endTime, s.limit)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ []katanaOrder
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	overrides := resolveIMFOverridesBestEffort(ctx, s.callAPI, wallet, marketFilter, opts...)

	var needFills []katanaOrder
	for _, o := range answ {
		if len(o.Fills) == 0 && !isZeroDecimal(o.ExecutedQuantity) {
			needFills = append(needFills, o)
		}
	}
	var fillsByOrder map[string][]katanaFill
	if len(needFills) > 0 {
		fillsByOrder, err = resolveMissingOrderFills(ctx, s.callAPI, wallet, marketFilter, s.startTime, s.endTime, needFills, mkts, opts...)
		if err != nil {
			return res, err
		}
	}

	out := s.convert.convertOrdersHistory(answ)

	var unknown unknownMarketTally
	for i := range out {
		o := answ[i]

		fills := o.Fills
		if len(fills) == 0 {
			fills = fillsByOrder[o.OrderId]
		}
		out[i].RealisedProfit, out[i].Fee, out[i].FeeAsset = aggregateFills(fills)

		if m, ok := effectiveMarket(mkts, overrides, o.Market); ok {
			if lev, lerr := imfToLeverage(m.InitialMarginFraction); lerr == nil {
				out[i].Leverage = lev
			}
		} else {
			unknown.add(o.Market)
		}
	}
	unknown.log("order(s)")

	sortDescendingByTime(out, func(o entity.Futures_OrdersHistory) int64 { return o.CreateTime })
	return out, nil
}

// maxPerOrderFillsFallback bounds the per-order GET /v1/fills calls below, so one history request
// cannot fan out past the User Data tier's 10/s rate limit.
const maxPerOrderFillsFallback = 20

// resolveMissingOrderFills backfills fills for executed orders that arrived without an inline
// `fills` array, so the action can aggregate their fee and PnL. The whole point is to never hand
// aggregateFills a partial fill set: an understated fee reads as a correct fee, which is worse than
// an honestly empty one.
//
//  1. Page the whole [start,end] window via fetchFillsPaged and group by orderId.
//  2. Reconcile each order against its bucket by summed quantity (quantityReconciles), not by mere
//     presence — an order whose fills straddle the window boundary or the page bound gets a PARTIAL
//     bucket, which a presence check would happily accept.
//  3. Anything that does not reconcile falls back to GET /v1/fills?orderId=..., which is unscoped by
//     time and therefore authoritative: its result REPLACES the partial bucket, never merges (a
//     merge would double-count fills present in both).
//  4. Past the fallback cap, error out naming the count rather than return partial money.
//
// mkts supplies each order's stepSize for the dust tolerance; an unknown market gets zero tolerance
// (exact match required) rather than an assumed one.
func resolveMissingOrderFills(
	ctx context.Context,
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error),
	wallet, market string,
	start, end *int64,
	needFills []katanaOrder,
	mkts map[string]Market,
	opts ...utils.RequestOption,
) (map[string][]katanaFill, error) {
	windowFills, err := fetchFillsPaged(ctx, callAPI, wallet, market, start, end, opts...)
	if err != nil {
		return nil, err
	}
	fillsByOrder := groupFillsByOrderID(windowFills)

	var unmatched []katanaOrder
	for _, o := range needFills {
		stepSize := ""
		if m, ok := mkts[o.Market]; ok {
			stepSize = m.StepSize
		}
		if !quantityReconciles(fillsByOrder[o.OrderId], o.ExecutedQuantity, stepSize) {
			unmatched = append(unmatched, o)
		}
	}
	if len(unmatched) == 0 {
		return fillsByOrder, nil
	}
	if len(unmatched) > maxPerOrderFillsFallback {
		return nil, fmt.Errorf("katana: %d orders' fills could not be reconciled from the windowed GET /v1/fills fetch and exceed the %d-order per-order fallback cap", len(unmatched), maxPerOrderFillsFallback)
	}

	for _, o := range unmatched {
		fills, ferr := fetchFillsForOrder(ctx, callAPI, wallet, o.OrderId, opts...)
		if ferr != nil {
			return nil, ferr
		}
		if len(fills) > 0 {
			fillsByOrder[o.OrderId] = fills
			continue
		}
		// Executed per the exchange, yet its own authoritative per-order fetch found no fills — a
		// data inconsistency an operator should see. Drop the partial bucket so the row reports an
		// honest empty rather than an understated total.
		delete(fillsByOrder, o.OrderId)
		log.Printf("katana: per-order GET /v1/fills?orderId=%s found no fills despite executedQuantity=%s; RealisedProfit/Fee left empty for this order", o.OrderId, o.ExecutedQuantity)
	}
	return fillsByOrder, nil
}

// quantityReconciles reports whether fills' summed Quantity reaches executedQuantity within one
// stepSize of dust. Every unreadable input fails CLOSED — an unparseable executedQuantity or
// stepSize routes to the authoritative per-order fetch instead of accepting a possibly partial
// bucket.
func quantityReconciles(fills []katanaFill, executedQuantity, stepSize string) bool {
	executed, err := parseBigFloat(executedQuantity)
	if err != nil {
		return false
	}

	sum := new(big.Float).SetPrec(bigPrec)
	for _, f := range fills {
		if v, ferr := parseBigFloat(f.Quantity); ferr == nil {
			sum.Add(sum, v)
		}
	}

	tolerance := new(big.Float).SetPrec(bigPrec)
	if t, terr := parseBigFloat(stepSize); terr == nil {
		tolerance = t
	}

	shortfall := new(big.Float).SetPrec(bigPrec).Sub(executed, sum)
	return shortfall.Cmp(tolerance) <= 0
}

func fetchFillsForOrder(
	ctx context.Context,
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error),
	wallet, orderID string,
	opts ...utils.RequestOption,
) ([]katanaFill, error) {
	nonce, _, err := newNonce()
	if err != nil {
		return nil, err
	}

	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v1/fills",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(utils.Params{
		"nonce":   nonce,
		"wallet":  wallet,
		"orderId": orderID,
	})

	data, _, err := callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}

	var fills []katanaFill
	if err := json.Unmarshal(data, &fills); err != nil {
		return nil, err
	}
	return fills, nil
}

// groupFillsByOrderID drops fills with no OrderId — liquidation/ADL fills omit it and cannot belong
// to a specific order's history row.
func groupFillsByOrderID(fills []katanaFill) map[string][]katanaFill {
	out := make(map[string][]katanaFill, len(fills))
	for _, f := range fills {
		if f.OrderId == "" {
			continue
		}
		out[f.OrderId] = append(out[f.OrderId], f)
	}
	return out
}

// aggregateFills sums realizedPnL and fee across an order's fills at bigPrec precision. Empty input
// returns "" everywhere (unknown), never a zero total. An individual unparseable field is genuinely
// absent on the wire ("omitted for open/liquidation actions") and contributes nothing rather than
// aborting the aggregation. feeAsset is "USDC" because Katana settles in vbUSDC and the fill object
// carries no fee-asset field of its own.
func aggregateFills(fills []katanaFill) (realisedProfit, fee, feeAsset string) {
	if len(fills) == 0 {
		return "", "", ""
	}

	pnlSum := new(big.Float).SetPrec(bigPrec)
	feeSum := new(big.Float).SetPrec(bigPrec)
	for _, f := range fills {
		if v, err := parseBigFloat(f.RealizedPnL); err == nil {
			pnlSum.Add(pnlSum, v)
		}
		if v, err := parseBigFloat(f.Fee); err == nil {
			feeSum.Add(feeSum, v)
		}
	}
	return formatDecimal8(pnlSum), formatDecimal8(feeSum), "USDC"
}

// katanaMaxPageLimit is the ceiling Katana documents for `limit` on both GET /v1/orders and
// GET /v1/fills ("Max results, 1-1,000").
const katanaMaxPageLimit = 1000

const katanaFillsPageCap = katanaMaxPageLimit

// maxFillsPagingPages caps one call at 10 x 1000 fills, so a deep or loosely-bounded window cannot
// turn a single history request into an unbounded scan against the 10/s tier.
const maxFillsPagingPages = 10

// fetchFillsPaged pages GET /v1/fills forward and returns every unique fill in the window. Two
// documented properties of the endpoint shape this loop:
//
//   - `fromId` is an INCLUSIVE boundary, so the fill it names comes back on the next page — hence
//     the `seen` set, without which that fill is double-counted.
//   - the response carries NO ordering guarantee, so the next cursor is latestFillID (greatest
//     `time` received), not the last array element. Off the array position, a non-ascending page
//     would point `fromId` at an arbitrary row and every later page would repeat, not advance.
//
// Hitting the page bound is logged once: a caller silently receiving truncated history is worse
// than a slower response that admits it.
func fetchFillsPaged(
	ctx context.Context,
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error),
	wallet, market string,
	start, end *int64,
	opts ...utils.RequestOption,
) ([]katanaFill, error) {
	var all []katanaFill
	seen := make(map[string]bool)
	fromID := ""
	truncated := true

	for page := 0; page < maxFillsPagingPages; page++ {
		nonce, _, err := newNonce()
		if err != nil {
			return nil, err
		}

		r := &utils.Request{
			Method:   http.MethodGet,
			Endpoint: "/v1/fills",
			SecType:  utils.SecTypeSigned,
		}
		r.SetParams(utils.Params{
			"nonce":  nonce,
			"wallet": wallet,
		})
		if market != "" {
			r.SetParam("market", market)
		}
		if start != nil {
			r.SetParam("start", strconv.FormatInt(*start, 10))
		}
		if end != nil {
			r.SetParam("end", strconv.FormatInt(*end, 10))
		}
		if fromID != "" {
			r.SetParam("fromId", fromID)
		}
		r.SetParam("limit", strconv.Itoa(katanaFillsPageCap))

		data, _, err := callAPI(ctx, r, opts...)
		if err != nil {
			return nil, err
		}

		var raw []katanaFill
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		if len(raw) == 0 {
			truncated = false
			break
		}

		for _, f := range raw {
			if seen[f.FillId] {
				continue
			}
			seen[f.FillId] = true
			all = append(all, f)
		}

		if len(raw) < katanaFillsPageCap {
			truncated = false
			break
		}
		fromID = latestFillID(raw)
	}

	if truncated {
		log.Printf("katana: fetchFillsPaged hit the %d-page bound for wallet=%s market=%q before the [start,end] window was exhausted; results may be incomplete", maxFillsPagingPages, wallet, market)
	}

	return all, nil
}

// latestFillID returns the FillId of the fill with the greatest Time — fetchFillsPaged's cursor,
// computed by time rather than array position because GET /v1/fills guarantees no wire order.
func latestFillID(fills []katanaFill) string {
	if len(fills) == 0 {
		return ""
	}
	latest := fills[0]
	for _, f := range fills[1:] {
		if f.Time > latest.Time {
			latest = f
		}
	}
	return latest.FillId
}

// clampLimit renders a caller-supplied Limit as a `limit` query value clamped to Katana's
// documented range. "" means omit the parameter and let Katana apply its own default of 50.
func clampLimit(limit *int64) string {
	if limit == nil {
		return ""
	}
	n := *limit
	if n <= 0 {
		return ""
	}
	if n > katanaMaxPageLimit {
		n = katanaMaxPageLimit
	}
	return strconv.FormatInt(n, 10)
}

// resolvedLimit is clampLimit's counterpart for fetchClosingFills, which needs a concrete number up
// front to know when it has collected enough closing rows rather than a query-string value.
func resolvedLimit(limit *int64, def int) int {
	if limit == nil || *limit <= 0 {
		return def
	}
	n := *limit
	if n > katanaMaxPageLimit {
		n = katanaMaxPageLimit
	}
	return int(n)
}

// applyHistoryPaging sets the market/start/end/limit params shared by ordersHistory and userTrades,
// and returns the resolved market filter for callers that also need it outside the query string.
// Page and Cursor have no Katana counterpart on these endpoints and are left unset rather than
// invented. positionsHistory pages via fetchClosingFills instead.
func applyHistoryPaging(r *utils.Request, symbol *string, start, end, limit *int64) string {
	marketFilter := ""
	if symbol != nil && strings.TrimSpace(*symbol) != "" {
		marketFilter = strings.TrimSpace(*symbol)
		r.SetParam("market", marketFilter)
	}
	if start != nil {
		r.SetParam("start", strconv.FormatInt(*start, 10))
	}
	if end != nil {
		r.SetParam("end", strconv.FormatInt(*end, 10))
	}
	if l := clampLimit(limit); l != "" {
		r.SetParam("limit", l)
	}
	return marketFilter
}

// sortDescendingByTime stable-sorts items newest-first. Katana's wire order for the history
// endpoints is undocumented, while pagers page backward by setting the next request's `end` from
// the current page's LAST row — correct only if that row is genuinely the oldest. Stable so rows
// sharing a timestamp keep a deterministic order across calls.
func sortDescendingByTime[T any](items []T, timeOf func(T) int64) {
	sort.SliceStable(items, func(i, j int) bool {
		return timeOf(items[i]) > timeOf(items[j])
	})
}
