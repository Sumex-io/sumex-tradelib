package whitebit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_amendOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol          *string
	orderID         *string
	clientOrderID   *string
	newAmount       *string
	newTotal        *string
	newPrice        *string
	activationPrice *string
}

func (s *spot_amendOrder) Symbol(symbol string) *spot_amendOrder {
	s.symbol = &symbol
	return s
}

func (s *spot_amendOrder) OrderID(orderID string) *spot_amendOrder {
	s.orderID = &orderID
	return s
}

func (s *spot_amendOrder) ClientOrderID(clientOrderID string) *spot_amendOrder {
	s.clientOrderID = &clientOrderID
	return s
}

func (s *spot_amendOrder) NewSize(amount string) *spot_amendOrder {
	s.newAmount = &amount
	return s
}

// нужно для stop-market buy (total вместо amount)
func (s *spot_amendOrder) NewTotal(total string) *spot_amendOrder {
	s.newTotal = &total
	return s
}

func (s *spot_amendOrder) NewPrice(price string) *spot_amendOrder {
	s.newPrice = &price
	return s
}

func (s *spot_amendOrder) ActivationPrice(price string) *spot_amendOrder {
	s.activationPrice = &price
	return s
}

func (s *spot_amendOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/order/modify",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	// market — обязательный в API
	if s.symbol != nil {
		m["market"] = *s.symbol
	}

	// идентификация ордера: либо orderId, либо clientOrderId
	if s.clientOrderID != nil {
		m["clientOrderId"] = *s.clientOrderID
	} else if s.orderID != nil {
		// WhiteBIT ожидает integer, но JSON и строку тоже переварит, оставим строкой.
		m["orderId"] = *s.orderID
	}

	// хотя бы один из этих параметров должен быть
	if s.newAmount != nil {
		m["amount"] = *s.newAmount
	}
	if s.newTotal != nil {
		m["total"] = *s.newTotal
	}
	if s.newPrice != nil {
		m["price"] = *s.newPrice
	}
	if s.activationPrice != nil {
		m["activationPrice"] = *s.activationPrice
	}

	if s.newAmount == nil && s.newTotal == nil && s.newPrice == nil && s.activationPrice == nil {
		return res, errors.New("nothing to modify: set at least one of amount/total/price/activationPrice")
	}

	// WhiteBIT v4 private — всегда JSON body
	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ spot_modifyOrderWB
	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	// оборачиваем в слайс, чтобы использовать общий конвертер
	return s.convert.convertSpotAmendOrder([]spot_modifyOrderWB{answ}), nil

}

// минимальная структура из ответа /api/v4/order/modify,
// нам важны только ID-ы, всё остальное уже есть в history/list.
type spot_modifyOrderWB struct {
	OrderID       int64  `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}
