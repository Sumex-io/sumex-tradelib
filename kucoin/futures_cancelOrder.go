package kucoin

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

	symbol  *string // пока не используем, но оставим на будущее (KuCoin иногда требует symbol в некоторых методах)
	orderID *string
	isTpSl  *bool
}

func (s *futures_cancelOrder) Symbol(symbol string) *futures_cancelOrder {
	s.symbol = &symbol
	return s
}

func (s *futures_cancelOrder) OrderID(orderID string) *futures_cancelOrder {
	s.orderID = &orderID
	return s
}

// IsTpSl(true) => отмена TP/SL (stop order), делаем через /api/v1/orders/multi-cancel
func (s *futures_cancelOrder) IsTpSl(v bool) *futures_cancelOrder {
	s.isTpSl = &v
	return s
}

func (s *futures_cancelOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	if s.orderID == nil || strings.TrimSpace(*s.orderID) == "" {
		return res, errors.New("kucoin futures_cancelOrder: OrderID is required")
	}
	oid := strings.TrimSpace(*s.orderID)

	isStop := s.isTpSl != nil && *s.isTpSl
	isStop = false
	// ---------------- TP/SL (stop order) cancel via multi-cancel ----------------
	// Docs: DELETE /api/v1/orders/multi-cancel with body {"orderIdsList":[...]} :contentReference[oaicite:1]{index=1}
	if isStop {
		r := &utils.Request{
			Method:   http.MethodDelete,
			Endpoint: "/api/v1/orders/multi-cancel",
			SecType:  utils.SecTypeSigned,
		}

		body := struct {
			OrderIdsList []string `json:"orderIdsList"`
		}{
			OrderIdsList: []string{oid},
		}

		b, e := json.Marshal(body)
		if e != nil {
			return res, e
		}
		r.BodyString = string(b)

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			return res, err
		}

		var answ struct {
			Code string `json:"code"`
			Msg  string `json:"msg"`
			Data []struct {
				OrderId   string `json:"orderId"`
				ClientOid string `json:"clientOid"`
				Code      string `json:"code"`
				Msg       string `json:"msg"`
			} `json:"data"`
		}

		if e := json.Unmarshal(data, &answ); e != nil {
			return res, e
		}
		if answ.Code != "" && answ.Code != "200000" {
			// если у тебя в проекте есть общий обработчик kucoin-ошибок — можно заменить на него
			return res, errors.New(string(data))
		}
		if len(answ.Data) == 0 {
			return res, errors.New("kucoin futures_cancelOrder: empty response data for multi-cancel")
		}
		if answ.Data[0].Code != "" && answ.Data[0].Code != "200" {
			return res, errors.New(string(data))
		}

		res = append(res, entity.PlaceOrder{
			OrderID:       oid,
			ClientOrderID: answ.Data[0].ClientOid,
			Ts:            time.Now().UTC().UnixMilli(),
		})
		return res, nil
	}

	// ---------------- normal order cancel ----------------
	r := &utils.Request{
		Method:   http.MethodDelete,
		Endpoint: "/api/v1/orders/{orderId}",
		SecType:  utils.SecTypeSigned,
	}
	r.Endpoint = strings.Replace(r.Endpoint, "{orderId}", oid, 1)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	// Обычно KuCoin отдаёт: { "code":"200000", "data":{ "cancelledOrderIds":[...] } }
	var answ struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			CancelledOrderIds []string `json:"cancelledOrderIds"`
		} `json:"data"`
	}

	if e := json.Unmarshal(data, &answ); e != nil {
		return res, e
	}
	if answ.Code != "" && answ.Code != "200000" {
		return res, errors.New(string(data))
	}

	res = append(res, entity.PlaceOrder{
		OrderID: oid,
		Ts:      time.Now().UTC().UnixMilli(),
	})
	return res, nil
}
