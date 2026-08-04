package gateio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_cancelOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	settle  *string
	orderID *string
	isTpSl  *bool // true => cancel price_orders (TP/SL trigger)
}

func (s *futures_cancelOrder) Settle(settle string) *futures_cancelOrder {
	s.settle = &settle
	return s
}

// IsTpSl(true) => отменяем trigger/TP/SL ордер (price_orders)
// IsTpSl(false) или не задан => обычный ордер (orders)
func (s *futures_cancelOrder) IsTpSl(v bool) *futures_cancelOrder {
	s.isTpSl = &v
	return s
}

func (s *futures_cancelOrder) OrderID(orderID string) *futures_cancelOrder {
	s.orderID = &orderID
	return s
}

func (s *futures_cancelOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	if s.orderID == nil || strings.TrimSpace(*s.orderID) == "" {
		return res, errors.New("gateio futures_cancelOrder: OrderID is required")
	}

	settleDefault := "usdt"
	if s.settle == nil || strings.TrimSpace(*s.settle) == "" {
		s.settle = &settleDefault
	}

	isTpSl := s.isTpSl != nil && *s.isTpSl

	endpoint := "/api/v4/futures/{settle}/orders/{order_id}"
	if isTpSl {
		// TP/SL (trigger) ордера живут в price_orders
		endpoint = "/api/v4/futures/{settle}/price_orders/{order_id}"
	}

	endpoint = strings.Replace(endpoint, "{settle}", *s.settle, 1)
	endpoint = strings.Replace(endpoint, "{order_id}", *s.orderID, 1)

	r := &utils.Request{
		Method:   http.MethodDelete,
		Endpoint: endpoint,
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// В Gate на cancel могут быть разные ответы в зависимости от типа ордера.
	// 1) иногда возвращают объект ордера (как на create) — тогда конвертер сработает
	// 2) иногда возвращают просто {"id":...} или пустой ответ
	// Сделаем best-effort:
	var answ futures_placeOrder_Response
	if e := json.Unmarshal(data, &answ); e == nil && answ.ID != 0 {
		return s.convert.convertPlaceOrder(answ), nil
	}

	// fallback: если вернули только id / либо формат другой — всё равно считаем отмену успешной
	// (если callAPI не вернул ошибку, значит код 2xx и Gate принял запрос)
	res = append(res, entity.PlaceOrder{
		OrderID: *s.orderID,
		Ts:      time.Now().UTC().UnixMilli(),
	})
	return res, nil
}
