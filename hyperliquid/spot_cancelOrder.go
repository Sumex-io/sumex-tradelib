package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_cancelOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol  *string
	orderID *string
}

func (s *spot_cancelOrder) Symbol(symbol string) *spot_cancelOrder {
	s.symbol = &symbol
	return s
}

func (s *spot_cancelOrder) OrderID(orderID string) *spot_cancelOrder {
	s.orderID = &orderID
	return s
}

func (s *spot_cancelOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return nil, fmt.Errorf("hyperliquid spot cancelOrder: symbol(assetId) is required")
	}
	if s.orderID == nil || strings.TrimSpace(*s.orderID) == "" {
		return nil, fmt.Errorf("hyperliquid spot cancelOrder: orderID is required")
	}

	asset, err := parseSpotAssetID(*s.symbol)
	if err != nil {
		return nil, err
	}

	oid, err := strconv.ParseInt(strings.TrimSpace(*s.orderID), 10, 64)
	if err != nil || oid <= 0 {
		return nil, fmt.Errorf("hyperliquid spot cancelOrder: invalid orderID %q", *s.orderID)
	}

	action := spotCancelOrderAction{
		Type: "cancel",
		Cancels: []spotCancelOrderItem{
			{
				A: asset,
				O: oid,
			},
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

	return s.convert.convertCancelOrderSpot(answ, oid), nil
}

type spotCancelOrderAction struct {
	Type    string                `json:"type"`
	Cancels []spotCancelOrderItem `json:"cancels"`
}

type spotCancelOrderItem struct {
	A int   `json:"a"`
	O int64 `json:"o"`
}
