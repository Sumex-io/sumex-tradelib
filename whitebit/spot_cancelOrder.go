package whitebit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ============== Cancel Spot Order =================

type spot_cancelOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol        *string
	orderID       *string
	clientOrderID *string
}

// Маркет (BTC_USDT и т.п.)
func (s *spot_cancelOrder) Symbol(symbol string) *spot_cancelOrder {
	s.symbol = &symbol
	return s
}

// Отмена по orderId
func (s *spot_cancelOrder) OrderID(orderID string) *spot_cancelOrder {
	s.orderID = &orderID
	return s
}

// Отмена по clientOrderId
func (s *spot_cancelOrder) ClientOrderID(clientOrderID string) *spot_cancelOrder {
	s.clientOrderID = &clientOrderID
	return s
}

func (s *spot_cancelOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	// По доке: нельзя одновременно и orderId, и clientOrderId
	if s.orderID == nil && s.clientOrderID == nil {
		return res, errors.New("either orderID or clientOrderID must be provided")
	}
	if s.orderID != nil && s.clientOrderID != nil {
		return res, errors.New("provide only one of orderID or clientOrderID, not both")
	}
	if s.symbol == nil {
		return res, errors.New("symbol (market) is required")
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/order/cancel",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"market": *s.symbol,
	}

	if s.clientOrderID != nil {
		m["clientOrderId"] = *s.clientOrderID
	} else if s.orderID != nil {
		m["orderId"] = *s.orderID
	}

	// WhiteBIT ждёт JSON-body, nonce/request добавит callAPI
	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ spot_modifyOrderWB
	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	// используем тот же конвертер, что и для amend
	return s.convert.convertSpotAmendOrder([]spot_modifyOrderWB{answ}), nil
}
