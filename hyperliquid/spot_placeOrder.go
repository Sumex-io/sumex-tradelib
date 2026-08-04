package hyperliquid

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_placeOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol        *string
	side          *entity.SideType
	size          *string
	price         *string
	orderType     *entity.OrderType
	clientOrderID *string

	positionSide *entity.PositionSideType
	tradeMode    *entity.MarginModeType
	tpPrice      *string
	slPrice      *string
}

func (s *spot_placeOrder) TradeMode(tradeMode entity.MarginModeType) *spot_placeOrder {
	s.tradeMode = &tradeMode
	return s
}

func (s *spot_placeOrder) SlPrice(slPrice string) *spot_placeOrder {
	s.slPrice = &slPrice
	return s
}

func (s *spot_placeOrder) TpPrice(tpPrice string) *spot_placeOrder {
	s.tpPrice = &tpPrice
	return s
}

func (s *spot_placeOrder) Symbol(symbol string) *spot_placeOrder {
	s.symbol = &symbol
	return s
}

func (s *spot_placeOrder) Side(side entity.SideType) *spot_placeOrder {
	s.side = &side
	return s
}

func (s *spot_placeOrder) Size(size string) *spot_placeOrder {
	s.size = &size
	return s
}

func (s *spot_placeOrder) Price(price string) *spot_placeOrder {
	s.price = &price
	return s
}

func (s *spot_placeOrder) OrderType(orderType entity.OrderType) *spot_placeOrder {
	s.orderType = &orderType
	return s
}

func (s *spot_placeOrder) ClientOrderID(clientOrderID string) *spot_placeOrder {
	s.clientOrderID = &clientOrderID
	return s
}

func (s *spot_placeOrder) PositionSide(positionSide entity.PositionSideType) *spot_placeOrder {
	s.positionSide = &positionSide
	return s
}

func (s *spot_placeOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return nil, fmt.Errorf("hyperliquid spot placeOrder: symbol(assetId) is required")
	}
	if s.side == nil {
		return nil, fmt.Errorf("hyperliquid spot placeOrder: side is required")
	}
	if s.size == nil || strings.TrimSpace(*s.size) == "" {
		return nil, fmt.Errorf("hyperliquid spot placeOrder: size is required")
	}

	asset, err := parseSpotAssetID(*s.symbol)
	if err != nil {
		return nil, err
	}
	spotIndex := asset - 10000

	ordType := entity.OrderTypeLimit
	if s.orderType != nil {
		ordType = *s.orderType
	}

	isBuy := strings.EqualFold(string(*s.side), string(entity.SideTypeBuy))
	tif := "Gtc"
	px := ""

	switch ordType {
	case entity.OrderTypeMarket:
		tif = "Ioc"

		if s.price != nil && strings.TrimSpace(*s.price) != "" {
			px = normalizeDecimalString(strings.TrimSpace(*s.price))
		} else {
			bestBid, bestAsk, book, err := s.getL2BookBySpotIndex(ctx, spotIndex, opts...)
			if err != nil {
				return nil, err
			}
			if bestBid == "" && bestAsk == "" {
				return nil, fmt.Errorf("hyperliquid: empty l2Book for @%d", spotIndex)
			}

			slipBps := int64(100)
			if isBuy {
				px, err = mulPxBps(bestAsk, slipBps)
			} else {
				px, err = mulPxBps(bestBid, -slipBps)
			}
			if err != nil {
				return nil, err
			}

			tick := inferTickFromBook(book)
			if tick == "" {
				ref := bestAsk
				if !isBuy {
					ref = bestBid
				}
				tick = tickFromDecimals(ref)
			}

			px, err = quantizeToTick(px, tick, isBuy)
			if err != nil {
				return nil, err
			}
		}

	default:
		if s.price == nil || strings.TrimSpace(*s.price) == "" {
			return nil, fmt.Errorf("hyperliquid spot placeOrder: price is required for LIMIT")
		}
		px = normalizeDecimalString(strings.TrimSpace(*s.price))
	}

	sz := normalizeDecimalString(strings.TrimSpace(*s.size))

	cloid := ""
	if s.clientOrderID != nil {
		cloid = normalizeCloid(*s.clientOrderID)
	}

	action := spotOrderAction{
		Type: "order",
		Orders: []spotOrder{
			{
				A: asset,
				B: isBuy,
				P: px,
				S: sz,
				R: false,
				T: spotOrderT{Limit: &spotOrderLimit{Tif: tif}},
				C: cloid,
			},
		},
		Grouping: "na",
	}

	b, _ := json.Marshal(action)

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/exchange",
		SecType:  utils.SecTypeSigned,
	}
	r.BodyString = string(b)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}

	answ := spot_placeOrderResponse{}
	if err := json.Unmarshal(data, &answ); err != nil {
		return nil, err
	}

	if len(answ.Response.Data.Statuses) > 0 {
		if e := strings.TrimSpace(answ.Response.Data.Statuses[0].Error); e != "" {
			return nil, &aPIError{Raw: data}
		}
	}

	res = s.convert.convertPlaceOrderSpot(answ)
	if len(res) > 0 && res[0].ClientOrderID == "" && cloid != "" {
		res[0].ClientOrderID = cloid
		res[0].Ts = time.Now().UTC().UnixMilli()
	}

	return res, nil
}

func parseSpotAssetID(symbol string) (int, error) {
	s := strings.TrimSpace(symbol)
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("hyperliquid: expected numeric spot assetId in Symbol(), got %q", symbol)
	}
	if n < 10000 {
		return 0, fmt.Errorf("hyperliquid: spot assetId must be >= 10000, got %d", n)
	}
	return n, nil
}

func normalizeCloid(in string) string {
	s := strings.TrimSpace(in)
	if s == "" {
		return ""
	}

	low := strings.ToLower(s)
	if strings.HasPrefix(low, "0x") {
		hexPart := low[2:]
		if len(hexPart) == 32 && isHexString(hexPart) {
			return "0x" + hexPart
		}
	}

	if len(low) == 32 && isHexString(low) {
		return "0x" + low
	}

	sum := md5.Sum([]byte(s))
	return "0x" + hex.EncodeToString(sum[:])
}

func isHexString(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func normalizeDecimalString(in string) string {
	s := strings.TrimSpace(in)
	if s == "" {
		return s
	}
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
		if s == "" {
			return "0"
		}
	}
	return s
}

type spotOrderAction struct {
	Type     string      `json:"type"`
	Orders   []spotOrder `json:"orders"`
	Grouping string      `json:"grouping"`
	Builder  interface{} `json:"builder,omitempty"`
}

type spotOrder struct {
	A int        `json:"a"`
	B bool       `json:"b"`
	P string     `json:"p"`
	S string     `json:"s"`
	R bool       `json:"r"`
	T spotOrderT `json:"t"`
	C string     `json:"c,omitempty"`
}

type spotOrderT struct {
	Limit   *spotOrderLimit   `json:"limit,omitempty"`
	Trigger *spotOrderTrigger `json:"trigger,omitempty"`
}

type spotOrderLimit struct {
	Tif string `json:"tif"`
}

type spotOrderTrigger struct {
	IsMarket  bool   `json:"isMarket"`
	TriggerPx string `json:"triggerPx"`
	Tpsl      string `json:"tpsl"`
}

type spot_placeOrderResponse struct {
	Status   string `json:"status"`
	Response struct {
		Type string `json:"type"`
		Data struct {
			Statuses []spotOrderStatus `json:"statuses"`
		} `json:"data"`
	} `json:"response"`
}

type spotOrderStatus struct {
	Oid     interface{} `json:"-"`
	Cloid   string      `json:"-"`
	Error   string      `json:"-"`
	Success bool        `json:"-"`
}

func (s *spotOrderStatus) UnmarshalJSON(b []byte) error {
	raw := strings.TrimSpace(string(b))
	if raw == "" || raw == "null" {
		return nil
	}

	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		if strings.EqualFold(strings.TrimSpace(v), "success") {
			s.Success = true
			return nil
		}
		if strings.TrimSpace(v) != "" {
			s.Error = v
		}
		return nil
	}

	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(b, &e); err == nil && strings.TrimSpace(e.Error) != "" {
		s.Error = e.Error
		return nil
	}

	var r struct {
		Resting *struct {
			Oid interface{} `json:"oid"`
			C   string      `json:"c,omitempty"`
		} `json:"resting"`
		Filled *struct {
			Oid interface{} `json:"oid"`
			C   string      `json:"c,omitempty"`
		} `json:"filled"`
		C string `json:"c,omitempty"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return err
	}
	if r.Resting != nil {
		s.Oid = r.Resting.Oid
		s.Cloid = r.Resting.C
	}
	if r.Filled != nil {
		s.Oid = r.Filled.Oid
		s.Cloid = r.Filled.C
	}
	if s.Cloid == "" {
		s.Cloid = r.C
	}
	return nil
}

type l2BookReq struct {
	Type string `json:"type"`
	Coin string `json:"coin"`
}

type l2BookLevel struct {
	Px string `json:"px"`
	Sz string `json:"sz"`
	N  int    `json:"n"`
}

type l2BookResp struct {
	Coin   string          `json:"coin"`
	Time   int64           `json:"time"`
	Levels [][]l2BookLevel `json:"levels"`
}

func (s *spot_placeOrder) getL2BookBySpotIndex(ctx context.Context, spotIndex int, opts ...utils.RequestOption) (bestBid, bestAsk string, resp l2BookResp, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/info",
		SecType:  utils.SecTypeNone,
	}

	coin := "@" + strconv.Itoa(spotIndex)
	b, _ := json.Marshal(l2BookReq{Type: "l2Book", Coin: coin})
	r.BodyString = string(b)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return "", "", resp, err
	}
	if string(data) == "null" {
		return "", "", resp, fmt.Errorf("hyperliquid: l2Book returned null for coin=%s", coin)
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", "", resp, err
	}

	if len(resp.Levels) >= 1 && len(resp.Levels[0]) > 0 {
		bestBid = resp.Levels[0][0].Px
	}
	if len(resp.Levels) >= 2 && len(resp.Levels[1]) > 0 {
		bestAsk = resp.Levels[1][0].Px
	}

	return bestBid, bestAsk, resp, nil
}

func inferTickFromBook(book l2BookResp) string {
	var candidates []string

	if len(book.Levels) >= 1 {
		if t := inferTickFromSide(book.Levels[0]); t != "" {
			candidates = append(candidates, t)
		}
	}
	if len(book.Levels) >= 2 {
		if t := inferTickFromSide(book.Levels[1]); t != "" {
			candidates = append(candidates, t)
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	var best *big.Rat
	for _, c := range candidates {
		r := new(big.Rat)
		if _, ok := r.SetString(c); !ok || r.Sign() <= 0 {
			continue
		}
		if best == nil || r.Cmp(best) < 0 {
			best = r
		}
	}
	if best == nil {
		return ""
	}
	return ratToFixedTrim(best, 12)
}

func inferTickFromSide(levels []l2BookLevel) string {
	if len(levels) < 2 {
		return ""
	}

	limit := len(levels)
	if limit > 30 {
		limit = 30
	}

	var minDiff *big.Rat
	var prev *big.Rat

	for i := 0; i < limit; i++ {
		cur := new(big.Rat)
		if _, ok := cur.SetString(levels[i].Px); !ok {
			continue
		}
		if prev != nil {
			d := new(big.Rat).Sub(prev, cur)
			if d.Sign() < 0 {
				d.Neg(d)
			}
			if d.Sign() > 0 {
				if minDiff == nil || d.Cmp(minDiff) < 0 {
					minDiff = d
				}
			}
		}
		prev = cur
	}

	if minDiff == nil || minDiff.Sign() <= 0 {
		return ""
	}
	return ratToFixedTrim(minDiff, 12)
}

func tickFromDecimals(px string) string {
	px = strings.TrimSpace(px)
	i := strings.IndexByte(px, '.')
	if i < 0 {
		return "1"
	}
	dec := len(px) - i - 1
	if dec <= 0 {
		return "1"
	}
	return "0." + strings.Repeat("0", dec-1) + "1"
}

func mulPxBps(px string, bps int64) (string, error) {
	px = strings.TrimSpace(px)
	if px == "" {
		return "", fmt.Errorf("empty px")
	}

	p := new(big.Rat)
	if _, ok := p.SetString(px); !ok {
		return "", fmt.Errorf("bad px %q", px)
	}

	num := big.NewInt(10000 + bps)
	den := big.NewInt(10000)
	k := new(big.Rat).SetFrac(num, den)

	p.Mul(p, k)
	return ratToFixedTrim(p, 18), nil
}

func quantizeToTick(px string, tick string, roundUp bool) (string, error) {
	px = strings.TrimSpace(px)
	tick = strings.TrimSpace(tick)
	if px == "" || tick == "" {
		return "", fmt.Errorf("empty px/tick")
	}

	p := new(big.Rat)
	if _, ok := p.SetString(px); !ok {
		return "", fmt.Errorf("bad px %q", px)
	}
	t := new(big.Rat)
	if _, ok := t.SetString(tick); !ok {
		return "", fmt.Errorf("bad tick %q", tick)
	}
	if t.Sign() <= 0 {
		return "", fmt.Errorf("tick must be > 0")
	}

	q := new(big.Rat).Quo(p, t)
	i := new(big.Int).Quo(q.Num(), q.Denom())

	if roundUp {
		rem := new(big.Int).Mod(q.Num(), q.Denom())
		if rem.Sign() != 0 {
			i.Add(i, big.NewInt(1))
		}
	}

	out := new(big.Rat).Mul(new(big.Rat).SetInt(i), t)
	dec := decimalsFromTick(tick)
	return ratToFixedTrim(out, dec), nil
}

func decimalsFromTick(tick string) int {
	tick = strings.TrimSpace(tick)
	if i := strings.IndexByte(tick, '.'); i >= 0 {
		return len(tick) - i - 1
	}
	return 0
}

func ratToFixedTrim(x *big.Rat, decimals int) string {
	s := x.FloatString(decimals)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}
