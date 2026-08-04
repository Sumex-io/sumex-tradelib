package whitebit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ================= Cancel Futures Order =================

type futures_cancelOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)

	symbol        *string
	orderID       *string
	clientOrderID *string
}

// Symbol (market), например "BTC_PERP"
func (s *futures_cancelOrder) Symbol(symbol string) *futures_cancelOrder {
	s.symbol = &symbol
	return s
}

// Биржевой orderId
func (s *futures_cancelOrder) OrderID(orderID string) *futures_cancelOrder {
	s.orderID = &orderID
	return s
}

// Наш clientOrderID (если хотим отменять по нему)
func (s *futures_cancelOrder) ClientOrderID(coid string) *futures_cancelOrder {
	s.clientOrderID = &coid
	return s
}

func (s *futures_cancelOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	// На WhiteBIT для /api/v4/order/cancel market обязателен
	// if s.symbol == nil || *s.symbol == "" {
	// 	return res, errors.New("symbol (market) is required for cancel order on WhiteBIT")
	// }

	if (s.orderID == nil || *s.orderID == "") && (s.clientOrderID == nil || *s.clientOrderID == "") {
		return res, errors.New("either orderID or clientOrderID must be provided for cancel order")
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/order/cancel",
		SecType:  utils.SecTypeSigned,
	}

	// m := utils.Params{
	// 	"market": *s.symbol, // пример: "BTC_PERP"
	// }
	m := utils.Params{}

	if s.symbol != nil {
		// Для фьючей WhiteBit формат и так "BTC_PERP", так что без конвертации
		m["market"] = *s.symbol
	}

	// WhiteBIT позволяет отменять либо по orderId, либо по clientOrderId
	if s.orderID != nil && *s.orderID != "" {
		// orderId на стороне биржи — число, но мы храним как string
		m["orderId"] = *s.orderID
	}
	if s.clientOrderID != nil && *s.clientOrderID != "" {
		m["clientOrderId"] = *s.clientOrderID
	}

	// У WhiteBIT всё подписывается через body (nonce + request + payload),
	// поэтому используем SetFormParams — request.go сам соберёт JSON и подпись.
	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// По документации /api/v4/order/cancel возвращает объект ордера.
	// Нам по факту достаточно orderId и clientOrderId.
	var answ struct {
		OrderId       int64  `json:"orderId"`
		ClientOrderId string `json:"clientOrderId"`
		Market        string `json:"market"`
		Side          string `json:"side"`
		Type          string `json:"type"`
		Status        string `json:"status"`
		// Остальные поля можно добавить при необходимости:
		// Timestamp float64 `json:"timestamp"`
		// ...
	}

	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	if answ.OrderId == 0 {
		return res, errors.New("empty orderId in cancel response")
	}

	// Конвертим в общий формат PlaceOrder (как и в других биржах)
	res = append(res, entity.PlaceOrder{
		OrderID:       strconv.FormatInt(answ.OrderId, 10),
		ClientOrderID: answ.ClientOrderId,
		// PositionID биржа в этом ответе не даёт, оставляем пустым.
		Ts: time.Now().UTC().UnixMilli(),
	})

	return res, nil
}
