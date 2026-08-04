package whitebit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_ordersHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64 // будем использовать как offset
	// orderID для WhiteBIT не применим, поэтому игнорируем
}

func (s *futures_ordersHistory) Symbol(symbol string) *futures_ordersHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_ordersHistory) StartTime(startTime int64) *futures_ordersHistory {
	s.startTime = &startTime
	return s
}

func (s *futures_ordersHistory) EndTime(endTime int64) *futures_ordersHistory {
	s.endTime = &endTime
	return s
}

func (s *futures_ordersHistory) Limit(limit int64) *futures_ordersHistory {
	s.limit = &limit
	return s
}

func (s *futures_ordersHistory) Page(page int64) *futures_ordersHistory {
	s.page = &page
	return s
}

// OrderID есть в общем интерфейсе, но WhiteBIT его не поддерживает в positions/history.
// Реализуем "пустой" сеттер, чтобы код компилировался и чейн не ломался.
func (s *futures_ordersHistory) OrderID(orderID string) *futures_ordersHistory {
	// WhiteBIT не умеет фильтровать по конкретному orderId в этом эндпоинте
	return s
}

func (s *futures_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersHistory, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/collateral-account/positions/history",
		SecType:  utils.SecTypeSigned,
	}

	body := utils.Params{}

	if s.symbol != nil {
		// Для фьючей WB формат market = BTC_PERP и т.п., он у нас как symbol уже такой
		body["market"] = *s.symbol
	}

	if s.limit != nil && *s.limit > 0 {
		body["limit"] = *s.limit
	}

	if s.page != nil && *s.page >= 0 {
		// В WhiteBIT это offset
		body["offset"] = *s.page
	}

	// startTime / endTime у нас в мс, WhiteBIT ждёт Unix seconds
	if s.startTime != nil && *s.startTime > 0 {
		body["startDate"] = *s.startTime / 1000
	}
	if s.endTime != nil && *s.endTime > 0 {
		body["endDate"] = *s.endTime / 1000
	}

	r.SetFormParams(body)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ []futures_positionsHistoryWB
	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertOrdersHistoryWB(answ), nil
}
