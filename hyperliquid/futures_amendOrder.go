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

type futures_amendOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol        *string
	side          *entity.SideType
	orderID       *string
	newSize       *string
	newPrice      *string
	clientOrderID *string
	reduce        *bool
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

func (s *futures_amendOrder) ClientOrderID(clientOrderID string) *futures_amendOrder {
	s.clientOrderID = &clientOrderID
	return s
}

func (s *futures_amendOrder) Reduce(reduce bool) *futures_amendOrder {
	s.reduce = &reduce
	return s
}

func (s *futures_amendOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return nil, fmt.Errorf("hyperliquid futures amendOrder: symbol is required")
	}
	if s.side == nil {
		return nil, fmt.Errorf("hyperliquid futures amendOrder: side is required")
	}
	if s.orderID == nil || strings.TrimSpace(*s.orderID) == "" {
		return nil, fmt.Errorf("hyperliquid futures amendOrder: orderID is required")
	}
	if s.newSize == nil || strings.TrimSpace(*s.newSize) == "" {
		return nil, fmt.Errorf("hyperliquid futures amendOrder: newSize is required")
	}
	if s.newPrice == nil || strings.TrimSpace(*s.newPrice) == "" {
		return nil, fmt.Errorf("hyperliquid futures amendOrder: newPrice is required")
	}

	asset, _, err := resolvePerpAsset(ctx, s.callAPI, strings.TrimSpace(*s.symbol), opts...)
	if err != nil {
		return nil, err
	}

	oid, err := strconv.ParseInt(strings.TrimSpace(*s.orderID), 10, 64)
	if err != nil || oid <= 0 {
		return nil, fmt.Errorf("hyperliquid futures amendOrder: invalid orderID %q", *s.orderID)
	}

	isBuy := strings.EqualFold(string(*s.side), string(entity.SideTypeBuy))
	newPx := normalizeDecimalString(strings.TrimSpace(*s.newPrice))
	newSz := normalizeDecimalString(strings.TrimSpace(*s.newSize))
	reduceOnly := s.reduce != nil && *s.reduce

	cloid := ""
	if s.clientOrderID != nil && strings.TrimSpace(*s.clientOrderID) != "" {
		cloid = normalizeCloid(*s.clientOrderID)
	}

	action := spotModifyOrderAction{
		Type: "modify",
		Oid:  oid,
		Order: spotOrder{
			A: asset,
			B: isBuy,
			P: newPx,
			S: newSz,
			R: reduceOnly,
			T: spotOrderT{
				Limit: &spotOrderLimit{Tif: "Gtc"},
			},
			C: cloid,
		},
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

	var answ struct {
		Status   string `json:"status"`
		Response struct {
			Type string `json:"type"`
			Data struct {
				Statuses []spotOrderStatus `json:"statuses"`
			} `json:"data"`
		} `json:"response"`
	}
	if err := json.Unmarshal(data, &answ); err != nil {
		return nil, err
	}

	if len(answ.Response.Data.Statuses) > 0 {
		if e := strings.TrimSpace(answ.Response.Data.Statuses[0].Error); e != "" {
			return nil, &aPIError{Raw: data}
		}
	}

	res = s.convert.convertPlaceOrder(spot_placeOrderResponse{
		Status: answ.Status,
		Response: struct {
			Type string `json:"type"`
			Data struct {
				Statuses []spotOrderStatus `json:"statuses"`
			} `json:"data"`
		}{
			Type: answ.Response.Type,
			Data: struct {
				Statuses []spotOrderStatus `json:"statuses"`
			}{
				Statuses: answ.Response.Data.Statuses,
			},
		},
	})

	if len(res) == 0 {
		res = append(res, entity.PlaceOrder{
			OrderID: strconv.FormatInt(oid, 10),
			Ts:      time.Now().UTC().UnixMilli(),
		})
	} else if res[0].OrderID == "" {
		res[0].OrderID = strconv.FormatInt(oid, 10)
	}

	if len(res) > 0 && res[0].ClientOrderID == "" && cloid != "" {
		res[0].ClientOrderID = cloid
		res[0].Ts = time.Now().UTC().UnixMilli()
	}

	return res, nil
}
