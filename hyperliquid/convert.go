package hyperliquid

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type account_converts struct{}

func (c *account_converts) convertAccountInfo(_ hyperliquidAccountInfo) (out entity.AccountInformation) {
	out.CanRead = true
	out.CanTrade = true
	out.CanTransfer = false
	out.PermSpot = true
	out.PermFutures = true
	out.UID = ""
	return out
}

type spot_converts struct{}

func (c *spot_converts) convertInstrumentsInfo(in spotMetaResponse) (out []entity.Spot_InstrumentsInfo) {
	if len(in.Universe) == 0 || len(in.Tokens) == 0 {
		return out
	}

	tokenByIndex := make(map[int]spotMetaToken, len(in.Tokens))
	for _, t := range in.Tokens {
		tokenByIndex[t.Index] = t
	}

	for _, u := range in.Universe {
		if len(u.Tokens) < 2 {
			continue
		}

		baseTok, okBase := tokenByIndex[u.Tokens[0]]
		quoteTok, okQuote := tokenByIndex[u.Tokens[1]]
		if !okBase || !okQuote {
			continue
		}

		symbol := u.Name
		if !strings.Contains(symbol, "/") {
			symbol = baseTok.Name + "/" + quoteTok.Name
		}

		// minQty := pow10Str(-baseTok.SzDecimals)
		sizePrec := strconv.Itoa(baseTok.SzDecimals)
		state := "LIVE"

		out = append(out, entity.Spot_InstrumentsInfo{
			Symbol: symbol,
			Base:   baseTok.Name,
			Quote:  quoteTok.Name,
			// MinQty:         minQty,
			MinNotional:    "",
			PricePrecision: "",
			SizePrecision:  sizePrec,
			State:          state,
			TokenId:        strconv.Itoa(10000 + u.Index),
		})
	}

	return out
}

func (c *spot_converts) convertSpotBalances(in spot_spotClearinghouseState) (out []entity.AssetsBalance) {
	for _, b := range in.Balances {
		if utils.StringToFloat(b.Total) == 0 && utils.StringToFloat(b.Hold) == 0 {
			continue
		}
		out = append(out, entity.AssetsBalance{
			Asset:   b.Coin,
			Balance: b.Total,
			Locked:  b.Hold,
		})
	}
	return out
}

func (c *spot_converts) convertPlaceOrderSpot(in spot_placeOrderResponse) (out []entity.PlaceOrder) {
	if len(in.Response.Data.Statuses) == 0 {
		return out
	}

	st := in.Response.Data.Statuses[0]
	if strings.TrimSpace(st.Error) != "" {
		return out
	}

	out = append(out, entity.PlaceOrder{
		OrderID:       stringifyHLID(st.Oid),
		ClientOrderID: strings.TrimSpace(st.Cloid),
		Ts:            time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *spot_converts) convertSpotOpenOrders(in []hlFrontendOpenOrder) (out []entity.Spot_OrdersList) {
	out = make([]entity.Spot_OrdersList, 0, len(in))

	for _, o := range in {
		assetID, ok := parseSpotAssetIDFromCoin(o.Coin)
		if !ok {
			continue
		}

		side := "BUY"
		if strings.EqualFold(strings.TrimSpace(o.Side), "A") {
			side = "SELL"
		}

		typ := strings.ToUpper(strings.TrimSpace(o.OrderType))
		if typ == "" {
			typ = "LIMIT"
		}

		size := strings.TrimSpace(o.OrigSz)
		if size == "" {
			size = strings.TrimSpace(o.Sz)
		}

		executed := "0"
		if strings.TrimSpace(o.OrigSz) != "" && strings.TrimSpace(o.Sz) != "" {
			if ex, err := decSub(o.OrigSz, o.Sz); err == nil {
				executed = ex
			}
		}

		out = append(out, entity.Spot_OrdersList{
			Symbol:        assetID,
			OrderID:       strconv.FormatInt(o.Oid, 10),
			ClientOrderID: o.clientOrderID(),
			Side:          side,
			Size:          size,
			ExecutedSize:  executed,
			Price:         strings.TrimSpace(o.LimitPx),
			Type:          typ,
			Status:        "NEW",
			CreateTime:    o.Timestamp,
			UpdateTime:    o.Timestamp,
		})
	}

	return out
}

func (c *spot_converts) convertCancelOrderSpot(_ spot_placeOrderResponse, oid int64) (out []entity.PlaceOrder) {
	out = append(out, entity.PlaceOrder{
		OrderID: strconv.FormatInt(oid, 10),
		Ts:      time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *spot_converts) convertSpotOrdersHistory(in []hlHistoricalOrder) (out []entity.Spot_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	out = make([]entity.Spot_OrdersHistory, 0, len(in))

	for _, item := range in {
		assetID, ok := parseSpotAssetIDFromHistoricalCoin(item.Order.Coin)
		if !ok {
			continue
		}

		status := strings.ToUpper(strings.TrimSpace(item.Status))
		if status != "FILLED" {
			continue
		}

		side := "BUY"
		if strings.EqualFold(strings.TrimSpace(item.Order.Side), "A") {
			side = "SELL"
		}

		orderType := strings.ToUpper(strings.TrimSpace(item.Order.OrderType))
		if orderType == "" {
			orderType = "LIMIT"
		}

		size := strings.TrimSpace(item.Order.OrigSz)
		if size == "" {
			size = strings.TrimSpace(item.Order.Sz)
		}

		createTime := item.Order.Timestamp
		updateTime := item.StatusTimestamp
		if updateTime == 0 {
			updateTime = createTime
		}

		out = append(out, entity.Spot_OrdersHistory{
			Symbol:        assetID,
			OrderID:       strconv.FormatInt(item.Order.Oid, 10),
			ClientOrderID: item.clientOrderID(),
			Side:          side,
			Size:          size,
			Price:         strings.TrimSpace(item.Order.LimitPx),
			ExecutedSize:  size,
			ExecutedPrice: strings.TrimSpace(item.Order.LimitPx),
			Type:          orderType,
			Status:        status,
			CreateTime:    createTime,
			UpdateTime:    updateTime,
		})
	}

	return out
}

func (c *spot_converts) convertSpotUserTrades(in []hlUserFill) (out []entity.Spot_UserTrades) {
	if len(in) == 0 {
		return out
	}

	out = make([]entity.Spot_UserTrades, 0, len(in))

	for _, item := range in {
		symbol, ok := parseSpotSymbolFromFillCoin(item.Coin)
		if !ok {
			continue
		}

		side := hlFillSide(item.Side)

		out = append(out, entity.Spot_UserTrades{
			TradeID:         stringifyHLID(item.Tid),
			OrderID:         stringifyHLID(item.Oid),
			Symbol:          symbol,
			Side:            side,
			Price:           strings.TrimSpace(item.Px),
			Qty:             strings.TrimSpace(item.Sz),
			QuoteQty:        hlFillQuoteQty(item),
			Commission:      strings.TrimSpace(item.Fee),
			CommissionAsset: strings.TrimSpace(item.FeeToken),
			RealisedProfit:  strings.TrimSpace(item.ClosedPnl),
			Buyer:           side == "BUY",
			Maker:           !item.Crossed,
			Crossed:         item.Crossed,
			Hash:            strings.TrimSpace(item.Hash),
			BuilderFee:      strings.TrimSpace(item.BuilderFee),
			Time:            item.Time,
		})
	}

	return out
}

type spotMetaResponse struct {
	Tokens   []spotMetaToken    `json:"tokens"`
	Universe []spotMetaUniverse `json:"universe"`
}

type spotMetaToken struct {
	Name        string `json:"name"`
	SzDecimals  int    `json:"szDecimals"`
	WeiDecimals int    `json:"weiDecimals"`
	Index       int    `json:"index"`
	TokenId     string `json:"tokenId"`
	IsCanonical bool   `json:"isCanonical"`
}

type spotMetaUniverse struct {
	Name        string `json:"name"`
	Tokens      []int  `json:"tokens"`
	Index       int    `json:"index"`
	IsCanonical bool   `json:"isCanonical"`
}

type futures_converts struct{}

func (c *futures_converts) convertInstrumentsInfo(in perpMetaResponse) (out []entity.Futures_InstrumentsInfo) {
	if len(in.Universe) == 0 {
		return out
	}

	for idx, u := range in.Universe {
		state := "LIVE"
		if u.IsDelisted {
			state = "OFFLINE"
		}

		base := strings.TrimSpace(u.Name)
		quote := "USDC"
		symbol := base + "/" + quote

		out = append(out, entity.Futures_InstrumentsInfo{
			Symbol:         symbol,
			Base:           base,
			Quote:          quote,
			MinQty:         pow10Str(-u.SzDecimals),
			MinNotional:    "",
			PricePrecision: "",
			SizePrecision:  strconv.Itoa(u.SzDecimals),
			State:          state,
			MaxLeverage:    utils.Int64ToString(int64(u.MaxLeverage)),
			// Multiplier:     "1",
			// ContractSize:   "1",
			IsSizeContract: false,
			TokenId:        strconv.Itoa(idx),
		})
	}

	return out
}

func (c *futures_converts) convertBalance(in futures_clearinghouseState) (out []entity.FuturesBalance) {
	unrealized := "0"
	for _, item := range in.AssetPositions {
		if strings.TrimSpace(item.Position.Coin) == "" {
			continue
		}
		if utils.StringToFloat(item.Position.Szi) == 0 && utils.StringToFloat(item.Position.UnrealizedPnl) == 0 {
			continue
		}
		unrealized = utils.FloatToStringAll(utils.StringToFloat(unrealized) + utils.StringToFloat(item.Position.UnrealizedPnl))
	}

	balance := in.CrossMarginSummary.AccountValue
	if strings.TrimSpace(balance) == "" {
		balance = in.MarginSummary.AccountValue
	}

	out = append(out, entity.FuturesBalance{
		Asset:            "USDC",
		Balance:          balance,
		Available:        in.Withdrawable,
		UnrealizedProfit: unrealized,
	})
	return out
}

func (c *futures_converts) convertLeverage(in futures_clearinghouseState, symbol string, marginMode *entity.MarginModeType) (out entity.Futures_Leverage) {
	for idx, item := range in.AssetPositions {
		coin := strings.TrimSpace(item.Position.Coin)
		if coin == "" {
			continue
		}
		if !futuresMatchSymbol(symbol, coin, idx) {
			continue
		}

		mode := strings.ToUpper(strings.TrimSpace(item.Position.Leverage.Type))
		// if marginMode != nil {
		// 	wantMode := strings.ToUpper(strings.TrimSpace(string(*marginMode)))
		// 	if wantMode != "" && mode != wantMode {
		// 		continue
		// 	}
		// }

		out.Symbol = coin + "/USDC"
		out.Leverage = utils.Int64ToString(item.Position.Leverage.Value)
		out.MarginMode = mode
		return out
	}

	return out
}

func (c *futures_converts) convertPositions(in futures_clearinghouseState) (out []entity.Futures_Positions) {
	for _, item := range in.AssetPositions {
		coin := strings.TrimSpace(item.Position.Coin)
		szi := strings.TrimSpace(item.Position.Szi)
		if coin == "" || utils.StringToFloat(szi) == 0 {
			continue
		}

		positionSide := "LONG"
		if utils.StringToFloat(szi) < 0 {
			positionSide = "SHORT"
		}

		marginMode := string(entity.MarginModeTypeCross)
		if strings.EqualFold(strings.TrimSpace(item.Position.Leverage.Type), "isolated") {
			marginMode = string(entity.MarginModeTypeIsolated)
		}

		size := absDecimalString(szi)
		notional := absDecimalString(strings.TrimSpace(item.Position.PositionValue))
		markPrice := ""
		if utils.StringToFloat(size) != 0 && utils.StringToFloat(notional) != 0 {
			markPrice = utils.FloatToStringAll(utils.StringToFloat(notional) / utils.StringToFloat(size))
		}

		out = append(out, entity.Futures_Positions{
			Symbol:           coin + "/USDC",
			PositionSide:     positionSide,
			PositionSize:     size,
			Leverage:         utils.Int64ToString(item.Position.Leverage.Value),
			EntryPrice:       item.Position.EntryPx,
			MarkPrice:        markPrice,
			UnRealizedProfit: item.Position.UnrealizedPnl,
			Notional:         notional,
			HedgeMode:        false,
			MarginMode:       marginMode,
			UpdateTime:       in.Time,
		})
	}
	return out
}

func (c *futures_converts) convertFuturesOrdersHistory(in []hlHistoricalOrder) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	out = make([]entity.Futures_OrdersHistory, 0, len(in))

	for _, item := range in {
		coin := strings.TrimSpace(item.Order.Coin)
		if coin == "" || strings.HasPrefix(coin, "@") {
			continue
		}

		status := strings.ToUpper(strings.TrimSpace(item.Status))
		if status != "FILLED" {
			continue
		}

		side := strings.ToUpper(strings.TrimSpace(item.Order.Side))
		positionSide := "LONG"
		if side == "A" {
			side = "SELL"
			positionSide = "SHORT"
		} else {
			side = "BUY"
			positionSide = "LONG"
		}

		orderType := strings.ToUpper(strings.TrimSpace(item.Order.OrderType))
		if orderType == "" {
			orderType = "LIMIT"
		}

		size := strings.TrimSpace(item.Order.OrigSz)
		if size == "" {
			size = strings.TrimSpace(item.Order.Sz)
		}

		createTime := item.Order.Timestamp
		updateTime := item.StatusTimestamp
		if updateTime == 0 {
			updateTime = createTime
		}

		tpOrder := false
		slOrder := false

		rawType := strings.ToUpper(strings.TrimSpace(item.Order.OrderType))
		tc := strings.ToUpper(strings.TrimSpace(item.Order.TriggerCondition))

		switch {
		case strings.Contains(rawType, "TAKE PROFIT"),
			strings.Contains(rawType, "TAKE"):
			tpOrder = true
			orderType = string(entity.OrderTypeTakeProfit)

		case strings.Contains(rawType, "STOP"):
			slOrder = true
			orderType = string(entity.OrderTypeStop)

		case item.Order.IsTrigger && (strings.Contains(tc, "PRICE ABOVE") ||
			strings.Contains(tc, "TP")):
			tpOrder = true
			orderType = string(entity.OrderTypeTakeProfit)

		case item.Order.IsTrigger && (strings.Contains(tc, "PRICE BELOW") ||
			strings.Contains(tc, "SL")):
			slOrder = true
			orderType = string(entity.OrderTypeStop)
		}
		if tpOrder || slOrder {
			if side == "SELL" {
				positionSide = "LONG"
			} else {
				positionSide = "SHORT"
			}
		}

		price := strings.TrimSpace(item.Order.LimitPx)
		executedPrice := price

		if item.Order.IsTrigger &&
			strings.TrimSpace(item.Order.TriggerPx) != "" &&
			strings.TrimSpace(item.Order.TriggerPx) != "0.0" {
			price = strings.TrimSpace(item.Order.TriggerPx)
		}

		out = append(out, entity.Futures_OrdersHistory{
			Symbol:        coin + "/USDC",
			OrderID:       strconv.FormatInt(item.Order.Oid, 10),
			ClientOrderID: item.clientOrderID(),
			Side:          side,
			PositionSide:  positionSide,
			PositionSize:  size,
			ExecutedSize:  size,
			Price:         price,
			ExecutedPrice: executedPrice,
			HedgeMode:     false,
			Type:          orderType,
			Status:        status,
			CreateTime:    createTime,
			UpdateTime:    updateTime,
			TpOrder:       tpOrder,
			SlOrder:       slOrder,
		})
	}

	return out
}

// Hyperliquid serves no positions history endpoint, so a position is replayed
// from the fills that built it: every fill carries the position size that
// preceded it, and a position ends the moment that size plus the fill reaches
// zero. Positions that are still open produce no row - they belong to the
// positions endpoint.
func (c *futures_converts) convertPositionsHistory(in []hlUserFill) (out []entity.Futures_PositionsHistory) {
	if len(in) == 0 {
		return out
	}

	out = make([]entity.Futures_PositionsHistory, 0, len(in))
	openPositions := make(map[string]*hlPositionAccumulator, len(in))

	for _, item := range hlFillsOldestFirst(in) {
		coin := strings.TrimSpace(item.Coin)
		if coin == "" || strings.HasPrefix(coin, "@") || strings.Contains(coin, "/") {
			continue
		}

		start := hlDecimal(item.StartPosition)
		size := hlDecimal(absDecimalString(item.Sz))
		if size.Sign() == 0 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(item.Side), "A") {
			size.Neg(size)
		}
		end := new(big.Rat).Add(start, size)

		position, ok := openPositions[coin]
		if !ok {
			position = newHLPositionAccumulator(coin, item, start, end)
			openPositions[coin] = position
		}
		position.addFill(item, start, end)

		if end.Sign() == 0 {
			out = appendHLPosition(out, position)
			delete(openPositions, coin)
			continue
		}

		// A flip ("Long > Short") closes the position and reopens the remainder
		// on the other side at the same price.
		if start.Sign() != 0 && end.Sign() != start.Sign() {
			out = appendHLPosition(out, position)
			openPositions[coin] = newHLPositionAccumulator(coin, item, new(big.Rat), end)
		}
	}

	return out
}

func (c *futures_converts) convertFuturesUserTrades(in []hlUserFill) (out []entity.Futures_UserTrades) {
	if len(in) == 0 {
		return out
	}

	out = make([]entity.Futures_UserTrades, 0, len(in))

	for _, item := range in {
		coin := strings.TrimSpace(item.Coin)
		if coin == "" || strings.HasPrefix(coin, "@") || strings.Contains(coin, "/") {
			continue
		}

		side := hlFillSide(item.Side)

		out = append(out, entity.Futures_UserTrades{
			TradeID:         stringifyHLID(item.Tid),
			OrderID:         stringifyHLID(item.Oid),
			Symbol:          coin + "/USDC",
			Side:            side,
			PositionSide:    hlFillPositionSide(item),
			Price:           strings.TrimSpace(item.Px),
			Qty:             strings.TrimSpace(item.Sz),
			QuoteQty:        hlFillQuoteQty(item),
			Commission:      strings.TrimSpace(item.Fee),
			CommissionAsset: strings.TrimSpace(item.FeeToken),
			RealisedProfit:  strings.TrimSpace(item.ClosedPnl),
			Buyer:           side == "BUY",
			Maker:           !item.Crossed,
			Crossed:         item.Crossed,
			Hash:            strings.TrimSpace(item.Hash),
			BuilderFee:      strings.TrimSpace(item.BuilderFee),
			Time:            item.Time,
		})
	}

	return out
}

func (c *futures_converts) convertPlaceOrder(in spot_placeOrderResponse) (out []entity.PlaceOrder) {
	if len(in.Response.Data.Statuses) == 0 {
		return out
	}

	st := in.Response.Data.Statuses[0]
	if strings.TrimSpace(st.Error) != "" {
		return out
	}

	out = append(out, entity.PlaceOrder{
		OrderID:       stringifyHLID(st.Oid),
		ClientOrderID: strings.TrimSpace(st.Cloid),
		Ts:            time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *futures_converts) convertCancelOrderSpot(_ spot_placeOrderResponse, oid int64) (out []entity.PlaceOrder) {
	out = append(out, entity.PlaceOrder{
		OrderID: strconv.FormatInt(oid, 10),
		Ts:      time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *futures_converts) convertFuturesOrderList(in []hlFrontendOpenOrderFutures) (out []entity.Futures_OrdersList) {
	out = make([]entity.Futures_OrdersList, 0, len(in))

	for _, o := range in {
		coin := strings.TrimSpace(o.Coin)
		if coin == "" || strings.HasPrefix(coin, "@") {
			continue
		}

		side := "BUY"
		if strings.EqualFold(strings.TrimSpace(o.Side), "A") {
			side = "SELL"
		}

		typ := strings.ToUpper(strings.TrimSpace(o.OrderType))
		if typ == "" {
			typ = "LIMIT"
		}

		tpOrder := false
		slOrder := false

		if o.IsTrigger {
			rawType := strings.ToUpper(strings.TrimSpace(o.OrderType))
			cond := strings.ToUpper(strings.TrimSpace(o.TriggerCondition))

			switch {
			case strings.Contains(rawType, "TAKE PROFIT"),
				strings.Contains(cond, "PRICE ABOVE"),
				strings.Contains(cond, "TP"),
				strings.Contains(cond, "TAKE"):
				typ = string(entity.OrderTypeTakeProfit)
				tpOrder = true

			case strings.Contains(rawType, "STOP"),
				strings.Contains(cond, "PRICE BELOW"),
				strings.Contains(cond, "SL"):
				typ = string(entity.OrderTypeStop)
				slOrder = true
			}
		}

		positionSide := ""
		switch {
		case tpOrder || slOrder:
			// TP/SL закрывает существующую позицию, поэтому side ордера
			// противоположна стороне позиции
			if side == "SELL" {
				positionSide = "LONG"
			} else {
				positionSide = "SHORT"
			}

		default:
			// обычная текущая логика для не-trigger/opening order
			if side == "SELL" {
				positionSide = "SHORT"
			} else {
				positionSide = "LONG"
			}
		}

		size := strings.TrimSpace(o.OrigSz)
		if size == "" {
			size = strings.TrimSpace(o.Sz)
		}

		executed := "0"
		if strings.TrimSpace(o.OrigSz) != "" && strings.TrimSpace(o.Sz) != "" {
			if ex, err := decSub(o.OrigSz, o.Sz); err == nil {
				executed = ex
			}
		}

		price := strings.TrimSpace(o.LimitPx)
		if o.IsTrigger && strings.TrimSpace(o.TriggerPx) != "" {
			price = strings.TrimSpace(o.TriggerPx)
		}

		out = append(out, entity.Futures_OrdersList{
			Symbol:        coin + "/USDC",
			OrderID:       strconv.FormatInt(o.Oid, 10),
			ClientOrderID: o.Cloid,
			PositionID:    "",
			Side:          side,
			PositionSide:  positionSide,
			PositionSize:  size,
			ExecutedSize:  executed,
			Price:         price,
			Leverage:      "",
			Type:          typ,
			Status:        "NEW",
			CreateTime:    o.Timestamp,
			UpdateTime:    o.Timestamp,
			MarginMode:    "",
			TpOrder:       tpOrder,
			SlOrder:       slOrder,
		})
	}

	return out
}

func (c *futures_converts) convertSetLeverage(symbol string, leverage int64) (out entity.Futures_Leverage) {
	out.Symbol = symbol
	out.Leverage = utils.Int64ToString(leverage)
	return out
}

type perpMetaResponse struct {
	Universe []perpMetaUniverse `json:"universe"`
}

type perpMetaUniverse struct {
	Name         string `json:"name"`
	SzDecimals   int    `json:"szDecimals"`
	MaxLeverage  int    `json:"maxLeverage"`
	OnlyIsolated bool   `json:"onlyIsolated,omitempty"`
	IsDelisted   bool   `json:"isDelisted,omitempty"`
}

func pow10Str(exp int) string {
	return utils.FloatToStringAll(math.Pow10(exp))
}

func stringifyHLID(v interface{}) string {
	if v == nil {
		return ""
	}

	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	case int64:
		return utils.Int64ToString(t)
	case uint64:
		return strconv.FormatUint(t, 10)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func decSub(a, b string) (string, error) {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return "", fmt.Errorf("empty a/b")
	}

	ra := new(big.Rat)
	if _, ok := ra.SetString(a); !ok {
		return "", fmt.Errorf("bad decimal %q", a)
	}
	rb := new(big.Rat)
	if _, ok := rb.SetString(b); !ok {
		return "", fmt.Errorf("bad decimal %q", b)
	}

	ra.Sub(ra, rb)

	s := ra.FloatString(18)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0", nil
	}
	return s, nil
}

func decMul(a, b string) (string, error) {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return "", fmt.Errorf("empty a/b")
	}

	ra := new(big.Rat)
	if _, ok := ra.SetString(a); !ok {
		return "", fmt.Errorf("bad decimal %q", a)
	}
	rb := new(big.Rat)
	if _, ok := rb.SetString(b); !ok {
		return "", fmt.Errorf("bad decimal %q", b)
	}

	ra.Mul(ra, rb)

	s := ra.FloatString(18)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0", nil
	}
	return s, nil
}

func hlFillQuoteQty(item hlUserFill) string {
	q, err := decMul(item.Px, item.Sz)
	if err != nil {
		return ""
	}
	return q
}

func hlFillSide(side string) string {
	if strings.EqualFold(strings.TrimSpace(side), "A") {
		return "SELL"
	}
	return "BUY"
}

func hlFillPositionSide(item hlUserFill) string {
	dir := strings.ToLower(strings.TrimSpace(item.Dir))
	if strings.Contains(dir, "short") {
		return "SHORT"
	}
	if strings.Contains(dir, "long") {
		return "LONG"
	}
	if strings.EqualFold(strings.TrimSpace(item.Side), "A") {
		return "SHORT"
	}
	return "LONG"
}

const (
	hlHistoryPriceDecimals  = 6
	hlHistorySizeDecimals   = 8
	hlHistoryAmountDecimals = 8
)

// hlPositionAccumulator collects the fills of one position until it closes.
type hlPositionAccumulator struct {
	coin           string
	positionID     string
	isLong         bool
	peakSize       *big.Rat
	closedSize     *big.Rat
	exitNotional   *big.Rat
	realisedProfit *big.Rat
	fee            *big.Rat
	openTime       int64
	closeTime      int64
}

// newHLPositionAccumulator starts a position at the given fill. A non-zero
// startPosition means the position was opened before the fills we were served,
// so its open time stays unknown and its entry has to be derived at close.
func newHLPositionAccumulator(coin string, item hlUserFill, start, end *big.Rat) *hlPositionAccumulator {
	position := &hlPositionAccumulator{
		coin:           coin,
		isLong:         start.Sign() > 0 || (start.Sign() == 0 && end.Sign() > 0),
		peakSize:       new(big.Rat).Abs(start),
		closedSize:     new(big.Rat),
		exitNotional:   new(big.Rat),
		realisedProfit: new(big.Rat),
		fee:            new(big.Rat),
		closeTime:      item.Time,
	}

	if start.Sign() == 0 {
		position.openTime = item.Time
	}

	return position
}

func (p *hlPositionAccumulator) addFill(item hlUserFill, start, end *big.Rat) {
	p.positionID = stringifyHLID(item.Tid)
	p.closeTime = item.Time
	p.fee.Add(p.fee, hlDecimal(item.Fee))
	p.trackPeak(start)
	p.trackPeak(end)

	if start.Sign() == 0 {
		return
	}

	filled := new(big.Rat).Sub(end, start)
	if filled.Sign() == start.Sign() {
		return
	}

	// The fill trades against the position, so it realises the smaller of the two:
	// anything beyond the open size belongs to the position a flip opens.
	closed := new(big.Rat).Abs(start)
	if size := new(big.Rat).Abs(filled); size.Cmp(closed) < 0 {
		closed = size
	}

	p.closedSize.Add(p.closedSize, closed)
	p.exitNotional.Add(p.exitNotional, new(big.Rat).Mul(closed, hlDecimal(item.Px)))
	p.realisedProfit.Add(p.realisedProfit, hlDecimal(item.ClosedPnl))
}

// trackPeak records the largest size the position held, which the row reports as
// the traded amount. Sizes on the other side belong to the position a flip opens.
func (p *hlPositionAccumulator) trackPeak(size *big.Rat) {
	if size.Sign() != 0 && (size.Sign() > 0) != p.isLong {
		return
	}

	if abs := new(big.Rat).Abs(size); abs.Cmp(p.peakSize) > 0 {
		p.peakSize = abs
	}
}

func (p *hlPositionAccumulator) toEntity() entity.Futures_PositionsHistory {
	// Fills carry no entry price, and the opening ones are usually older than the
	// window we were served, so the entry of the closed amount comes out of the PnL.
	positionSide := "LONG"
	entryNotional := new(big.Rat).Sub(p.exitNotional, p.realisedProfit)
	if !p.isLong {
		positionSide = "SHORT"
		entryNotional = new(big.Rat).Add(p.exitNotional, p.realisedProfit)
	}

	return entity.Futures_PositionsHistory{
		Symbol:              p.coin + "/USDC",
		PositionID:          p.positionID,
		PositionSide:        positionSide,
		PositionAmt:         hlDecimalString(p.peakSize, hlHistorySizeDecimals),
		ExecutedPositionAmt: hlDecimalString(p.closedSize, hlHistorySizeDecimals),
		AvgPrice:            hlDecimalString(new(big.Rat).Quo(entryNotional, p.closedSize), hlHistoryPriceDecimals),
		ExecutedAvgPrice:    hlDecimalString(new(big.Rat).Quo(p.exitNotional, p.closedSize), hlHistoryPriceDecimals),
		RealisedProfit:      hlDecimalString(p.realisedProfit, hlHistoryAmountDecimals),
		Fee:                 hlDecimalString(p.fee, hlHistoryAmountDecimals),
		CreateTime:          p.openTime,
		UpdateTime:          p.closeTime,
	}
}

// appendHLPosition drops positions that closed without realising anything,
// which is what a zero-size fill on a flat position would produce.
func appendHLPosition(out []entity.Futures_PositionsHistory, position *hlPositionAccumulator) []entity.Futures_PositionsHistory {
	if position == nil || position.closedSize.Sign() == 0 {
		return out
	}

	return append(out, position.toEntity())
}

// hlFillsOldestFirst replays fills in the order the position was built. `/info`
// answers newest first, so reversing keeps fills that share a millisecond in
// their original sequence, which the stable sort then preserves.
func hlFillsOldestFirst(in []hlUserFill) []hlUserFill {
	out := make([]hlUserFill, len(in))
	copy(out, in)

	if len(out) > 1 && out[0].Time > out[len(out)-1].Time {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Time < out[j].Time })

	return out
}

func hlDecimal(value string) *big.Rat {
	parsed := new(big.Rat)
	if _, ok := parsed.SetString(strings.TrimSpace(value)); !ok {
		return new(big.Rat)
	}

	return parsed
}

func hlDecimalString(value *big.Rat, decimals int) string {
	s := value.FloatString(decimals)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	// An amount below the last kept decimal must not come out as "-0".
	if s == "" || s == "-" || s == "-0" {
		return "0"
	}

	return s
}

func parseSpotSymbolFromFillCoin(coin string) (symbol string, ok bool) {
	c := strings.TrimSpace(coin)
	if strings.HasPrefix(c, "@") {
		return parseSpotAssetIDFromHistoricalCoin(c)
	}
	if strings.Contains(c, "/") {
		return c, true
	}
	return "", false
}

func absDecimalString(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "-") {
		return strings.TrimPrefix(s, "-")
	}
	return s
}
