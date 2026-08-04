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

type spot_amendOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol        *string
	orderID       *string
	side          *entity.SideType
	newSize       *string
	newPrice      *string
	clientOrderID *string
}

func (s *spot_amendOrder) Symbol(symbol string) *spot_amendOrder {
	s.symbol = &symbol
	return s
}

func (s *spot_amendOrder) OrderID(orderID string) *spot_amendOrder {
	s.orderID = &orderID
	return s
}

func (s *spot_amendOrder) Side(side entity.SideType) *spot_amendOrder {
	s.side = &side
	return s
}

func (s *spot_amendOrder) NewSize(newSize string) *spot_amendOrder {
	s.newSize = &newSize
	return s
}

func (s *spot_amendOrder) NewPrice(newPrice string) *spot_amendOrder {
	s.newPrice = &newPrice
	return s
}

func (s *spot_amendOrder) ClientOrderID(clientOrderID string) *spot_amendOrder {
	s.clientOrderID = &clientOrderID
	return s
}

func (s *spot_amendOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return nil, fmt.Errorf("hyperliquid spot amendOrder: symbol(assetId) is required")
	}
	if s.orderID == nil || strings.TrimSpace(*s.orderID) == "" {
		return nil, fmt.Errorf("hyperliquid spot amendOrder: orderID is required")
	}
	if s.side == nil {
		return nil, fmt.Errorf("hyperliquid spot amendOrder: side is required")
	}
	if s.newSize == nil || strings.TrimSpace(*s.newSize) == "" {
		return nil, fmt.Errorf("hyperliquid spot amendOrder: newSize is required")
	}
	if s.newPrice == nil || strings.TrimSpace(*s.newPrice) == "" {
		return nil, fmt.Errorf("hyperliquid spot amendOrder: newPrice is required")
	}

	asset, err := parseSpotAssetID(*s.symbol)
	if err != nil {
		return nil, err
	}

	oid, err := strconv.ParseInt(strings.TrimSpace(*s.orderID), 10, 64)
	if err != nil || oid <= 0 {
		return nil, fmt.Errorf("hyperliquid spot amendOrder: invalid orderID %q", *s.orderID)
	}

	isBuy := strings.EqualFold(string(*s.side), string(entity.SideTypeBuy))
	newPx := normalizeDecimalString(strings.TrimSpace(*s.newPrice))
	newSz := normalizeDecimalString(strings.TrimSpace(*s.newSize))

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
			R: false,
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

type spotModifyOrderAction struct {
	Type  string    `json:"type"`
	Oid   int64     `json:"oid"`
	Order spotOrder `json:"order"`
}
