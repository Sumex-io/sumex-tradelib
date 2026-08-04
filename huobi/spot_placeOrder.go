package huobi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	positionSide  *entity.PositionSideType
	tradeMode     *entity.MarginModeType
	tpPrice       *string
	slPrice       *string
	uid           *string
}

func (s *spot_placeOrder) UID(uid string) *spot_placeOrder {
	s.uid = &uid
	return s
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
		Endpoint: "/v1/order/orders/place",
		SecType:  utils.SecTypeSigned,
	}

	sideDef := ""
	m := utils.Params{}

	if s.symbol != nil {
		m["symbol"] = strings.ToLower(*s.symbol)
	}

	if s.side != nil {
		// m["side"] = strings.ToLower(string(*s.side))
		sideDef = strings.ToLower(string(*s.side))
	}

	if s.price != nil {
		m["price"] = *s.price
	}
	if s.orderType != nil {
		m["type"] = fmt.Sprintf("%s-%s", sideDef, strings.ToLower(string(*s.orderType)))
		// m["type"] = strings.ToLower(string(*s.orderType))
		// if *s.orderType == entity.OrderTypeMarket {
		// 	// m["price"] = "0"
		// 	m["time_in_force"] = "ioc"
		// }
	}

	if s.size != nil {
		m["amount"] = *s.size
	}

	if s.uid != nil {
		m["account-id"] = *s.uid
	}

	if s.clientOrderID != nil {
		m["client-order-id"] = *s.clientOrderID
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result        string `json:"data"`
		ClientOrderId string `json:"clientOrderId"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	// return s.convert.convertPlaceOrder(answ), nil
	return []entity.PlaceOrder{{OrderID: answ.Result, ClientOrderID: answ.ClientOrderId, Ts: time.Now().UTC().UnixMilli()}}, nil
}

type placeOrder_Response struct {
	Account_id      string `json:"account-id"`
	Client_order_id string `json:"client-order-id"`
}
