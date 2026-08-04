package whitebit

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

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

// ====== Do ======

func (s *spot_placeOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	r := &utils.Request{
		Method:  http.MethodPost,
		SecType: utils.SecTypeSigned,
	}

	// Определяем endpoint по типу ордера
	// MARKET -> /api/v4/order/market
	// LIMIT  -> /api/v4/order/new
	if s.orderType != nil && *s.orderType == entity.OrderTypeMarket {
		r.Endpoint = "/api/v4/order/market"
	} else {
		// по умолчанию считаем, что это лимит
		r.Endpoint = "/api/v4/order/new"
	}

	m := utils.Params{}

	if s.symbol != nil {
		// На WhiteBIT спотовый market = "BTC_USDT"
		m["market"] = *s.symbol
	}

	if s.side != nil {
		// биржа ждёт "buy" / "sell"
		m["side"] = strings.ToLower(string(*s.side))
	}

	if s.size != nil {
		// amount — количество base (stock), как в доке:
		// "Amount of stock currency to buy or sell"
		m["amount"] = *s.size
	}

	// Цена нужна только для лимитных
	if s.orderType != nil && *s.orderType == entity.OrderTypeLimit && s.price != nil {
		m["price"] = *s.price
	}

	if s.clientOrderID != nil {
		m["clientOrderId"] = *s.clientOrderID
	}

	// Укладываем всё в body (nonce + request + поля)
	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := spot_placeOrderResponseWB{}
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertPlaceOrder(answ), nil
}

// минимальная структура под ответ WhiteBIT spot order
type spot_placeOrderResponseWB struct {
	OrderID       int64  `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}
