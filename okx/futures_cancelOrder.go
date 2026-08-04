package okx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_cancelOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol  *string
	orderID *string

	isTpSl *bool
}

func (s *futures_cancelOrder) IsTpSl(v bool) *futures_cancelOrder {
	s.isTpSl = &v
	return s
}

func (s *futures_cancelOrder) Symbol(symbol string) *futures_cancelOrder {
	s.symbol = &symbol
	return s
}

func (s *futures_cancelOrder) OrderID(orderID string) *futures_cancelOrder {
	s.orderID = &orderID
	return s
}

func (s *futures_cancelOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	isAlgo := s.isTpSl != nil && *s.isTpSl

	r := &utils.Request{
		Method:  http.MethodPost,
		SecType: utils.SecTypeSigned,
	}

	if !isAlgo {
		m := utils.Params{}
		r.Endpoint = "/api/v5/trade/cancel-order"
		// instId требуется и для обычной отмены, и для cancel-algos
		if s.symbol != nil {
			m["instId"] = *s.symbol
		}

		if s.orderID != nil {

			m["ordId"] = *s.orderID
		}

		r.SetFormParams(m)
	} else {
		r.Endpoint = "/api/v5/trade/cancel-algos"
		m := utils.Params{}
		type cancelAlgoReq struct {
			InstId string `json:"instId"`
			AlgoId string `json:"algoId"`
		}

		if s.symbol == nil || s.orderID == nil {
			return res, errors.New("instId and algoId required")
		}

		body := []cancelAlgoReq{
			{
				InstId: *s.symbol,
				AlgoId: *s.orderID,
			},
		}

		b, err := json.Marshal(body)
		if err != nil {
			return res, err
		}

		m["cancel-algos"] = string(b)
		r.SetFormParams(m)
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []placeOrder_Response `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ.Result) == 0 {
		return res, errors.New("Zero Answer")
	}

	if answ.Result[0].SCode != "0" {
		return res, errors.New(answ.Result[0].SMsg)
	}

	// Важно:
	// - cancel-algos в ответе обычно возвращает algoId/algoClOrdId,
	//   а наш placeOrder_Response ожидает ordId/clOrdId.
	// - Чтобы не трогать конвертер и сохранить "тот же ответ", нормализуем:
	if isAlgo {
		tmp := make([]placeOrder_Response, 0, len(answ.Result))
		for _, it := range answ.Result {
			// если в it.OrdId уже лежит algoId — ок,
			// если в JSON пришло "algoId" и распарсилось в OrdId (зависит от тэгов в placeOrder_Response) — тоже ок.
			tmp = append(tmp, placeOrder_Response{
				ClOrdId: it.ClOrdId,
				OrdId:   it.OrdId,
				Tag:     it.Tag,
				Ts:      it.Ts,
				SCode:   it.SCode,
				SMsg:    it.SMsg,
			})
		}
		return s.convert.convertPlaceOrder(tmp), nil
	}

	return s.convert.convertPlaceOrder(answ.Result), nil
}
