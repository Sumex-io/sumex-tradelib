package bitget

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_cancelOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol  *string
	orderID *string
	isTpSl  *bool
	isTp    *bool
	isSl    *bool
}

func (s *futures_cancelOrder) IsTpSl(v bool) *futures_cancelOrder {
	s.isTpSl = &v
	return s
}

func (s *futures_cancelOrder) IsTp(v bool) *futures_cancelOrder {
	s.isTp = &v
	return s
}

func (s *futures_cancelOrder) IsSl(v bool) *futures_cancelOrder {
	s.isSl = &v
	return s
}

func (s *futures_cancelOrder) OrderID(orderID string) *futures_cancelOrder {
	s.orderID = &orderID
	return s
}

func (s *futures_cancelOrder) Symbol(symbol string) *futures_cancelOrder {
	s.symbol = &symbol
	return s
}

func (s *futures_cancelOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	isPlane := s.isTpSl != nil && *s.isTpSl

	if (s.isTp != nil && *s.isTp == true) || (s.isSl != nil && *s.isSl == true) {
		isPlane = true
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v2/mix/order/cancel-order",
		SecType:  utils.SecTypeSigned,
	}

	if !isPlane {

		m := utils.Params{"productType": "USDT-FUTURES"}

		if s.symbol != nil {
			m["symbol"] = *s.symbol
		}

		if s.orderID != nil {
			m["orderId"] = *s.orderID
		}

		r.SetFormParams(m)

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			return res, err
		}

		var answ struct {
			Result futures_placeOrder_Response `json:"data"`
		}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}
		res = s.convert.convertPlaceOrder(answ.Result)
		return res, nil
	} else {
		r.Endpoint = "/api/v2/mix/order/cancel-plan-order"

		m := utils.Params{
			"productType": "USDT-FUTURES",
			"marginCoin":  "USDT",
			// "planType":    "profit_plan",
		}

		if s.symbol != nil {
			m["symbol"] = *s.symbol
		}
		if s.orderID != nil {
			m["orderId"] = *s.orderID
		}

		if s.isTp != nil && *s.isTp == true {
			m["planType"] = "profit_plan"
		}

		if s.isSl != nil && *s.isSl == true {
			m["planType"] = "loss_plan"
		}

		r.SetFormParams(m)

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			return res, err
		}

		var answ futures_cancelPlan_Wrapper
		if err := json.Unmarshal(data, &answ); err != nil {
			return res, err
		}

		// Более надёжно чем msg
		if answ.Code != "" && answ.Code != "00000" {
			return res, errors.New(string(data))
		}

		// Если есть failureList — это точно ошибка отмены
		if len(answ.Data.FailureList) > 0 || len(answ.Data.SuccessList) == 0 {
			return res, errors.New(string(data))
		}

		// Если мы отменяем конкретный orderId — он должен оказаться в successList
		if s.orderID != nil {
			for _, it := range answ.Data.SuccessList {
				if it.OrderId == *s.orderID {
					res = append(res, entity.PlaceOrder{
						OrderID:       it.OrderId,
						ClientOrderID: it.ClientOid,
						Ts:            time.Now().UnixMilli(),
					})
					return res, nil
				}
			}
			// success, но ордера нет в списке успеха -> считаем, что не отменили то, что просили
			return res, errors.New(string(data))
		}

		// Если orderId не задан (массовая отмена) — просто вернём все successList
		for _, it := range answ.Data.SuccessList {
			res = append(res, entity.PlaceOrder{
				OrderID:       it.OrderId,
				ClientOrderID: it.ClientOid,
				Ts:            time.Now().UnixMilli(),
			})
		}

		// Если вообще пусто — тоже вернём пусто (не ошибка)
		return res, nil
	}

}

type futures_cancelPlan_Wrapper struct {
	Code string                  `json:"code"`
	Msg  string                  `json:"msg"`
	Data futures_cancelPlan_Data `json:"data"`
}

type futures_cancelPlan_Data struct {
	SuccessList []futures_cancelPlan_Item `json:"successList"`
	FailureList []futures_cancelPlan_Fail `json:"failureList"`
}

type futures_cancelPlan_Item struct {
	OrderId   string `json:"orderId"`
	ClientOid string `json:"clientOid"`
}

type futures_cancelPlan_Fail struct {
	OrderId   string `json:"orderId"`
	ClientOid string `json:"clientOid"`
	ErrorCode string `json:"errorCode"`
	ErrorMsg  string `json:"errorMsg"`
}
