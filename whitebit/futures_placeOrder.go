package whitebit

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
	reduce        *bool

	// TP/SL flags (для отдельного trigger-order)
	tpOrder *bool
	slOrder *bool
}

// ========= сеттеры =========

func (s *futures_placeOrder) Reduce(reduce bool) *futures_placeOrder {
	s.reduce = &reduce
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

// TP/SL setters
func (s *futures_placeOrder) TpOrder(v bool) *futures_placeOrder {
	s.tpOrder = &v
	return s
}

func (s *futures_placeOrder) SlOrder(v bool) *futures_placeOrder {
	s.slOrder = &v
	return s
}

// ========= основная логика =========

func (s *futures_placeOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	// минимальная валидация TP/SL флагов: одновременно нельзя
	isTP := s.tpOrder != nil && *s.tpOrder
	isSL := s.slOrder != nil && *s.slOrder
	if isTP && isSL {
		return res, errors.New("whitebit futures_placeOrder: TpOrder and SlOrder cannot both be true")
	}

	// Проверка базовых обязательных полей для обычного flow (оставляем как было),
	// но для TP/SL мы допускаем, что caller может не передать price/size и т.п. —
	// биржа вернёт ошибку, если что. Тем не менее минимально требуем для обычного ордера:
	if !isTP && !isSL {
		if s.symbol == nil || s.side == nil || s.size == nil {
			return res, errors.New("symbol, side and size are required")
		}
	}

	// Определяем тип ордера:
	// - если явно указан MARKET — маркет
	// - иначе считаем LIMIT (для лимита обязательно нужна цена)
	orderType := entity.OrderTypeLimit
	if s.orderType != nil {
		orderType = *s.orderType
	}

	r := &utils.Request{
		Method:  http.MethodPost,
		SecType: utils.SecTypeSigned,
	}

	m := utils.Params{}

	// Общие поля (используются и в обычном, и в trigger-ветке)
	if s.symbol != nil {
		market := *s.symbol // у нас в instruments вернулся тот же формат, что и в market (BTC_PERP и т.п.)
		m["market"] = market
	}
	if s.side != nil {
		m["side"] = strings.ToLower(string(*s.side))
	}
	if s.size != nil {
		m["amount"] = *s.size
	}
	if s.clientOrderID != nil {
		m["clientOrderId"] = *s.clientOrderID
	}
	if s.positionSide != nil {
		// WhiteBIT ожидает: LONG / SHORT / BOTH
		m["positionSide"] = strings.ToUpper(string(*s.positionSide))
	}
	if s.reduce != nil && *s.reduce == true {
		m["reduceOnly"] = true
	}

	// --- TP/SL as separate trigger-market order (conditional) ---
	if isTP || isSL {
		// Trigger Market Order endpoint
		r.Endpoint = "/api/v4/order/collateral/trigger-market"

		// В нашем унифицированном интерфейсе Price() = trigger price
		if s.price != nil {
			m["activation_price"] = *s.price
		}

		// WhiteBIT принимает также optional поля stopLoss/takeProfit, но мы не используем attach-поле.
		// Просто отправляем trigger-market с activation_price и остальными базовыми полями.

		r.SetFormParams(m)

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			return res, err
		}

		// Ответ для trigger-market совпадает по форме с place order (поля orderId и т.п.)
		var answ futures_placeOrderResponse
		if err = json.Unmarshal(data, &answ); err != nil {
			return res, err
		}

		if answ.OrderID == 0 {
			return res, errors.New("empty orderId in whitebit response")
		}

		return s.convert.convertPlaceOrder(answ), nil
	}
	// --- end TP/SL branch ---

	// Выбор endpoint'а по типу ордера (обычная логика)
	switch orderType {
	case entity.OrderTypeMarket:
		// Маркет ордер фьючерсов / маржи
		r.Endpoint = "/api/v4/order/collateral/market"
	case entity.OrderTypeLimit:
		// Лимитный ордер фьючерсов / маржи
		r.Endpoint = "/api/v4/order/collateral/limit"
		if s.price == nil {
			return res, errors.New("price is required for LIMIT order")
		}
		m["price"] = *s.price
	default:
		return res, errors.New("unsupported order type for whitebit futures")
	}

	// WhiteBIT всё принимает в body (JSON)
	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// Ответ для collateral market / limit — один объект ордера
	var answ futures_placeOrderResponse
	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	// На всякий случай проверим, что orderId не пустой
	if answ.OrderID == 0 {
		return res, errors.New("empty orderId in whitebit response")
	}

	return s.convert.convertPlaceOrder(answ), nil
}

// внутренняя структура ответа whitebit для place order
type futures_placeOrderResponse struct {
	OrderID       int64   `json:"orderId"`
	ClientOrderID string  `json:"clientOrderId"`
	Market        string  `json:"market"`
	Side          string  `json:"side"`
	Type          string  `json:"type"` // "market" / "limit"
	Timestamp     float64 `json:"timestamp"`
	// дополнительные поля нам сейчас не нужны, но можно добавить при необходимости
	// DealMoney  string  `json:"dealMoney"`
	// DealStock  string  `json:"dealStock"`
	// Amount     string  `json:"amount"`
	// Left       string  `json:"left"`
	// DealFee    string  `json:"dealFee"`
	// Status     string  `json:"status"`
	// PositionSide string `json:"positionSide"`
}

// маленький помощник для float64 timestamp -> ms
func tsFloatToMillis(ts float64) int64 {
	if ts <= 0 {
		return time.Now().UTC().UnixMilli()
	}
	// timestamp приходит как "1595792396.165973" -> секунды с микросекундами
	sec := int64(ts)
	msFrac := int64((ts - float64(sec)) * 1000.0)
	return sec*1000 + msFrac
}
