package weex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_placeOrder struct {
	callAPI  func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	brokerID string
	convert  spot_converts

	symbol        *string
	side          *entity.SideType
	size          *string
	price         *string
	orderType     *entity.OrderType
	clientOrderID *string
	positionSide  *entity.PositionSideType
	tradeMode     *entity.MarginModeType
	tpPrice       *string
	slPrice       *string
}

func (s *spot_placeOrder) TradeMode(tradeMode entity.MarginModeType) *spot_placeOrder {
	s.tradeMode = &tradeMode
	return s
}

func (s *spot_placeOrder) SlPrice(slPrice string) *spot_placeOrder {
	s.slPrice = &slPrice
	return s
}

func (s *spot_placeOrder) TpPrice(tpPrice string) *spot_placeOrder {
	s.tpPrice = &tpPrice
	return s
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

func (s *spot_placeOrder) PositionSide(positionSide entity.PositionSideType) *spot_placeOrder {
	s.positionSide = &positionSide
	return s
}

func (s *spot_placeOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v3/order",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.side != nil {
		m["side"] = strings.ToUpper(string(*s.side))
	}

	if s.orderType != nil {
		m["type"] = strings.ToUpper(string(*s.orderType))
	}

	if s.size != nil {
		m["quantity"] = *s.size
	}

	if s.price != nil {
		m["price"] = *s.price
	}

	if s.clientOrderID != nil {
		m["newClientOrderId"] = *s.clientOrderID
	}

	if s.orderType != nil && strings.ToUpper(string(*s.orderType)) == "LIMIT" {
		m["timeInForce"] = "GTC"
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ placeOrder_Response
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if answ.OrderId == 0 {
		return res, errors.New("zero answer")
	}

	return s.convert.convertPlaceOrder(answ), nil
}

type placeOrder_Response struct {
	Symbol        string `json:"symbol"`
	OrderId       int64  `json:"orderId"`
	ClientOrderId string `json:"clientOrderId"`
	TransactTime  int64  `json:"transactTime"`
}
