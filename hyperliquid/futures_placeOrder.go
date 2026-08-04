package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_placeOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol        *string
	side          *entity.SideType
	size          *string
	price         *string
	orderType     *entity.OrderType
	clientOrderID *string
	positionSide  *entity.PositionSideType
	marginMode    *entity.MarginModeType

	reduce  *bool
	tpOrder *bool
	slOrder *bool
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

func (s *futures_placeOrder) MarginMode(marginMode entity.MarginModeType) *futures_placeOrder {
	s.marginMode = &marginMode
	return s
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

func (s *futures_placeOrder) OrderType(orderType entity.OrderType) *futures_placeOrder {
	s.orderType = &orderType
	return s
}

func (s *futures_placeOrder) ClientOrderID(clientOrderID string) *futures_placeOrder {
	s.clientOrderID = &clientOrderID
	return s
}

func (s *futures_placeOrder) PositionSide(positionSide entity.PositionSideType) *futures_placeOrder {
	s.positionSide = &positionSide
	return s
}

func (s *futures_placeOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return nil, fmt.Errorf("hyperliquid futures placeOrder: symbol is required")
	}
	if s.side == nil {
		return nil, fmt.Errorf("hyperliquid futures placeOrder: side is required")
	}
	if s.size == nil || strings.TrimSpace(*s.size) == "" {
		return nil, fmt.Errorf("hyperliquid futures placeOrder: size is required")
	}

	asset, coin, err := resolvePerpAsset(ctx, s.callAPI, strings.TrimSpace(*s.symbol), opts...)
	if err != nil {
		return nil, err
	}

	isBuy := strings.EqualFold(string(*s.side), string(entity.SideTypeBuy))
	size := normalizeDecimalString(strings.TrimSpace(*s.size))

	cloid := ""
	if s.clientOrderID != nil && strings.TrimSpace(*s.clientOrderID) != "" {
		cloid = normalizeCloid(*s.clientOrderID)
	}

	reduceOnly := s.reduce != nil && *s.reduce
	isTP := s.tpOrder != nil && *s.tpOrder
	isSL := s.slOrder != nil && *s.slOrder

	if isTP && isSL {
		return nil, fmt.Errorf("hyperliquid futures placeOrder: tpOrder and slOrder cannot both be true")
	}

	grouping := "na"
	order := spotOrder{
		A: asset,
		B: isBuy,
		S: size,
		R: reduceOnly,
		C: cloid,
	}

	if isTP || isSL {
		if s.price == nil || strings.TrimSpace(*s.price) == "" {
			return nil, fmt.Errorf("hyperliquid futures placeOrder: price is required for tp/sl order")
		}

		triggerPx := normalizeDecimalString(strings.TrimSpace(*s.price))

		// Hyperliquid требует валидный p даже для trigger isMarket=true.
		// Для SELL p должен быть <= triggerPx, для BUY p должен быть >= triggerPx.
		_, _, book, err := s.getPerpL2Book(ctx, coin, opts...)
		if err != nil {
			return nil, err
		}

		tick := inferTickFromBook(book)
		if tick == "" {
			tick = tickFromDecimals(triggerPx)
		}

		// Небольшой "защитный" отступ от triggerPx.
		// SELL -> чуть ниже trigger
		// BUY  -> чуть выше trigger
		bps := int64(10)
		if !isBuy {
			bps = -10
		}

		px, err := mulPxBps(triggerPx, bps)
		if err != nil {
			return nil, err
		}

		px, err = quantizeToTick(px, tick, isBuy)
		if err != nil {
			return nil, err
		}

		order.P = px
		order.R = true
		order.T = spotOrderT{
			Trigger: &spotOrderTrigger{
				IsMarket:  true,
				TriggerPx: triggerPx,
				Tpsl:      map[bool]string{true: "tp", false: "sl"}[isTP],
			},
		}
	} else {
		ordType := entity.OrderTypeLimit
		if s.orderType != nil {
			ordType = *s.orderType
		}

		switch ordType {
		case entity.OrderTypeMarket:
			bestBid, bestAsk, book, err := s.getPerpL2Book(ctx, coin, opts...)
			if err != nil {
				return nil, err
			}
			if bestBid == "" && bestAsk == "" {
				return nil, fmt.Errorf("hyperliquid futures placeOrder: empty l2Book for %s", coin)
			}

			slipBps := int64(10)

			pxRef := bestAsk
			bps := slipBps
			if !isBuy {
				pxRef = bestBid
				bps = -slipBps
			}

			px, err := mulPxBps(pxRef, bps)
			if err != nil {
				return nil, err
			}

			tick := inferTickFromBook(book)
			if tick == "" {
				tick = tickFromDecimals(pxRef)
			}

			px, err = quantizeToTick(px, tick, isBuy)
			if err != nil {
				return nil, err
			}

			order.P = px
			order.T = spotOrderT{Limit: &spotOrderLimit{Tif: "Ioc"}}

		case entity.OrderTypeStop:
			if s.price == nil || strings.TrimSpace(*s.price) == "" {
				return nil, fmt.Errorf("hyperliquid futures placeOrder: price is required for stop order")
			}

			triggerPx := normalizeDecimalString(strings.TrimSpace(*s.price))

			_, _, book, err := s.getPerpL2Book(ctx, coin, opts...)
			if err != nil {
				return nil, err
			}

			tick := inferTickFromBook(book)
			if tick == "" {
				tick = tickFromDecimals(triggerPx)
			}

			bps := int64(10)
			if !isBuy {
				bps = -10
			}

			px, err := mulPxBps(triggerPx, bps)
			if err != nil {
				return nil, err
			}

			px, err = quantizeToTick(px, tick, isBuy)
			if err != nil {
				return nil, err
			}

			order.P = px
			order.T = spotOrderT{
				Trigger: &spotOrderTrigger{
					IsMarket:  true,
					TriggerPx: triggerPx,
					Tpsl:      "sl",
				},
			}

		case entity.OrderTypeTakeProfit:
			if s.price == nil || strings.TrimSpace(*s.price) == "" {
				return nil, fmt.Errorf("hyperliquid futures placeOrder: price is required for take profit order")
			}

			triggerPx := normalizeDecimalString(strings.TrimSpace(*s.price))

			_, _, book, err := s.getPerpL2Book(ctx, coin, opts...)
			if err != nil {
				return nil, err
			}

			tick := inferTickFromBook(book)
			if tick == "" {
				tick = tickFromDecimals(triggerPx)
			}

			bps := int64(10)
			if !isBuy {
				bps = -10
			}

			px, err := mulPxBps(triggerPx, bps)
			if err != nil {
				return nil, err
			}

			px, err = quantizeToTick(px, tick, isBuy)
			if err != nil {
				return nil, err
			}

			order.P = px
			order.T = spotOrderT{
				Trigger: &spotOrderTrigger{
					IsMarket:  true,
					TriggerPx: triggerPx,
					Tpsl:      "tp",
				},
			}

		default:
			if s.price == nil || strings.TrimSpace(*s.price) == "" {
				return nil, fmt.Errorf("hyperliquid futures placeOrder: price is required for limit order")
			}
			order.P = normalizeDecimalString(strings.TrimSpace(*s.price))
			order.T = spotOrderT{Limit: &spotOrderLimit{Tif: "Gtc"}}
		}
	}

	action := spotOrderAction{
		Type:     "order",
		Orders:   []spotOrder{order},
		Grouping: grouping,
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

	res = s.convert.convertPlaceOrder(answ)
	if len(res) > 0 && res[0].ClientOrderID == "" && cloid != "" {
		res[0].ClientOrderID = cloid
		res[0].Ts = time.Now().UTC().UnixMilli()
	}

	return res, nil
}

func resolvePerpAsset(
	ctx context.Context,
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error),
	symbol string,
	opts ...utils.RequestOption,
) (asset int, coin string, err error) {
	sym := strings.TrimSpace(symbol)
	if sym == "" {
		return 0, "", fmt.Errorf("hyperliquid futures: empty symbol")
	}

	meta, err := getPerpMeta(ctx, callAPI, opts...)
	if err != nil {
		return 0, "", err
	}

	if n, e := strconv.Atoi(sym); e == nil && n >= 0 {
		if n >= len(meta.Universe) {
			return 0, "", fmt.Errorf("hyperliquid futures: perp asset id %d out of range", n)
		}
		return n, strings.TrimSpace(meta.Universe[n].Name), nil
	}

	want := strings.ToUpper(sym)
	wantNoSlash := strings.ReplaceAll(want, "/", "")

	for idx, u := range meta.Universe {
		base := strings.ToUpper(strings.TrimSpace(u.Name))
		pair := base + "/USDC"
		pairNoSlash := base + "USDC"

		if want == base || want == pair || wantNoSlash == pairNoSlash {
			return idx, strings.TrimSpace(u.Name), nil
		}
	}

	return 0, "", fmt.Errorf("hyperliquid futures: cannot resolve perp asset for symbol %q", symbol)
}

func getPerpMeta(
	ctx context.Context,
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error),
	opts ...utils.RequestOption,
) (resp perpMetaResponse, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/info",
		SecType:  utils.SecTypeNone,
	}

	b, _ := json.Marshal(map[string]interface{}{
		"type": "meta",
	})
	r.BodyString = string(b)

	data, _, err := callAPI(ctx, r, opts...)
	if err != nil {
		return resp, err
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

func (s *futures_placeOrder) getPerpL2Book(ctx context.Context, coin string, opts ...utils.RequestOption) (bestBid, bestAsk string, resp l2BookResp, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/info",
		SecType:  utils.SecTypeNone,
	}

	b, _ := json.Marshal(l2BookReq{Type: "l2Book", Coin: coin})
	r.BodyString = string(b)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return "", "", resp, err
	}
	if string(data) == "null" {
		return "", "", resp, fmt.Errorf("hyperliquid futures placeOrder: l2Book returned null for coin=%s", coin)
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
