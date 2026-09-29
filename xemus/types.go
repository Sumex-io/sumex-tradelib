package xemus

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Wire types for perp-api responses. perp-api passes Orderly's `data` through unwrapped, with
// Orderly's field names, JSON numbers for every quantity and price, and Unix-millisecond
// timestamps. Numbers are decoded as num (a json.Number that tolerates null) so no value is ever
// routed through float64 on its way to the string fields of the entity package.

// num is a JSON number kept as its literal text. null and absent both decode to "".
type num string

func (n *num) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		*n = ""
		return nil
	}
	// Tolerate a quoted number: Orderly documents a few fields as strings in one place and numbers
	// in another (mark_price, amount), and a type flip must not blank a whole panel.
	if strings.HasPrefix(s, `"`) {
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		*n = num(strings.TrimSpace(str))
		return nil
	}
	var jn json.Number
	if err := json.Unmarshal(b, &jn); err != nil {
		return err
	}
	*n = num(jn.String())
	return nil
}

// String renders the number as a plain decimal. JavaScript serialises magnitudes below 1e-6 in
// exponent form ("1e-7"), which is how a small tick size reaches this connector, and consumers of
// the entity strings expect plain decimals.
func (n num) String() string {
	return plainDecimal(string(n))
}

func (n num) int64() int64 {
	v, err := strconv.ParseInt(string(n), 10, 64)
	if err != nil {
		f, ferr := strconv.ParseFloat(string(n), 64)
		if ferr != nil {
			return 0
		}
		return int64(f)
	}
	return v
}

// flexString decodes a JSON string or number into its text. Orderly's docs type client_order_id
// as a number and its SDK as a string, and perp-api accepts either (review doc open item).
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		*f = ""
		return nil
	}
	if strings.HasPrefix(s, `"`) {
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		*f = flexString(str)
		return nil
	}
	*f = flexString(s)
	return nil
}

// flexBool decodes true/false as well as Orderly's 0/1 flags (is_maker).
type flexBool bool

func (f *flexBool) UnmarshalJSON(b []byte) error {
	switch strings.Trim(strings.TrimSpace(string(b)), `"`) {
	case "true", "1":
		*f = true
	default:
		*f = false
	}
	return nil
}

// ===============RESPONSES=================

type accountInfoResponse struct {
	AccountID   string `json:"account_id"`
	MaxLeverage num    `json:"max_leverage"`
}

type streamSignatureResponse struct {
	AccountID  string `json:"accountId"`
	OrderlyKey string `json:"orderlyKey"`
	Timestamp  num    `json:"timestamp"`
	Signature  string `json:"signature"`
	URL        string `json:"url"`
}

type symbolRow struct {
	Symbol      string `json:"symbol"`
	QuoteTick   num    `json:"quote_tick"`
	BaseMin     num    `json:"base_min"`
	BaseTick    num    `json:"base_tick"`
	MinNotional num    `json:"min_notional"`
	BaseIMR     num    `json:"base_imr"`
	Status      string `json:"status"`
}

type positionRow struct {
	Symbol           string `json:"symbol"`
	PositionQty      num    `json:"position_qty"`
	AverageOpenPrice num    `json:"average_open_price"`
	MarkPrice        num    `json:"mark_price"`
	Leverage         num    `json:"leverage"`
	MarginMode       string `json:"margin_mode"`
	Timestamp        num    `json:"timestamp"`
}

type positionsResponse struct {
	Rows                 []positionRow `json:"rows"`
	FreeCollateral       num           `json:"free_collateral"`
	TotalCollateralValue num           `json:"total_collateral_value"`
}

type orderRow struct {
	OrderID              num        `json:"order_id"`
	AlgoOrderID          num        `json:"algo_order_id"`
	ClientOrderID        flexString `json:"client_order_id"`
	Symbol               string     `json:"symbol"`
	Side                 string     `json:"side"`
	Type                 string     `json:"type"`
	Status               string     `json:"status"`
	Price                num        `json:"price"`
	Quantity             num        `json:"quantity"`
	Executed             num        `json:"executed"`
	TotalExecutedQty     num        `json:"total_executed_quantity"`
	AverageExecutedPrice num        `json:"average_executed_price"`
	TotalFee             num        `json:"total_fee"`
	FeeAsset             string     `json:"fee_asset"`
	ReduceOnly           bool       `json:"reduce_only"`
	MarginMode           string     `json:"margin_mode"`
	CreatedTime          num        `json:"created_time"`
	UpdatedTime          num        `json:"updated_time"`
}

type ordersResponse struct {
	Rows []orderRow `json:"rows"`
}

// algoOrderRow is one node of an algo order tree. A TP_SL / POSITIONAL_TP_SL root carries its
// TAKE_PROFIT / STOP_LOSS legs in child_orders; a BRACKET root carries one POSITIONAL_TP_SL,
// which carries the legs; a STOP root carries none.
//
// is_activated is false on a leg switched off by an edit (Orderly's way of cancelling one leg of
// two); absent, the leg is taken as live.
type algoOrderRow struct {
	AlgoOrderID     num            `json:"algo_order_id"`
	RootAlgoOrderID num            `json:"root_algo_order_id"`
	ClientOrderID   flexString     `json:"client_order_id"`
	Symbol          string         `json:"symbol"`
	AlgoType        string         `json:"algo_type"`
	Side            string         `json:"side"`
	Type            string         `json:"type"`
	Quantity        num            `json:"quantity"`
	TriggerPrice    num            `json:"trigger_price"`
	Price           num            `json:"price"`
	IsTriggered     flexBool       `json:"is_triggered"`
	IsActivated     *flexBool      `json:"is_activated"`
	AlgoStatus      string         `json:"algo_status"`
	ReduceOnly      bool           `json:"reduce_only"`
	MarginMode      string         `json:"margin_mode"`
	CreatedTime     num            `json:"created_time"`
	UpdatedTime     num            `json:"updated_time"`
	ChildOrders     []algoOrderRow `json:"child_orders"`
}

type algoOrdersResponse struct {
	Rows []algoOrderRow `json:"rows"`
}

type tradeRow struct {
	ID                num        `json:"id"`
	OrderID           num        `json:"order_id"`
	Symbol            string     `json:"symbol"`
	Side              string     `json:"side"`
	ExecutedPrice     num        `json:"executed_price"`
	ExecutedQuantity  num        `json:"executed_quantity"`
	Fee               num        `json:"fee"`
	FeeAsset          string     `json:"fee_asset"`
	RealizedPnl       num        `json:"realized_pnl"`
	IsMaker           flexBool   `json:"is_maker"`
	ExecutedTimestamp num        `json:"executed_timestamp"`
	ClientOrderID     flexString `json:"client_order_id"`
}

type tradesResponse struct {
	Rows []tradeRow `json:"rows"`
}

// positionHistoryRow's margin_mode is typed "CROSS" | "ISOLATED" | 1 | 0 by Orderly's SDK, so it is
// decoded as flexString and normalised in the mapper.
type positionHistoryRow struct {
	PositionID            num        `json:"position_id"`
	PositionStatus        string     `json:"position_status"`
	Symbol                string     `json:"symbol"`
	AvgOpenPrice          num        `json:"avg_open_price"`
	AvgClosePrice         num        `json:"avg_close_price"`
	MaxPositionQty        num        `json:"max_position_qty"`
	ClosedPositionQty     num        `json:"closed_position_qty"`
	Side                  string     `json:"side"`
	TradingFee            num        `json:"trading_fee"`
	AccumulatedFundingFee num        `json:"accumulated_funding_fee"`
	RealizedPnl           num        `json:"realized_pnl"`
	OpenTimestamp         num        `json:"open_timestamp"`
	CloseTimestamp        num        `json:"close_timestamp"`
	LastUpdateTime        num        `json:"last_update_time"`
	Leverage              num        `json:"leverage"`
	MarginMode            flexString `json:"margin_mode"`
}

// rows may be null for an account with no history (perp-api review, testnet run 2026-09-24);
// ranging over a nil slice is already safe.
type positionHistoryResponse struct {
	Rows []positionHistoryRow `json:"rows"`
}

type leverageRow struct {
	Symbol     string `json:"symbol"`
	Leverage   num    `json:"leverage"`
	MarginMode string `json:"margin_mode"`
}

type marginModeRow struct {
	Symbol            string `json:"symbol"`
	DefaultMarginMode string `json:"default_margin_mode"`
}

type placeOrderResponse struct {
	OrderID       num        `json:"order_id"`
	ClientOrderID flexString `json:"client_order_id"`
}
