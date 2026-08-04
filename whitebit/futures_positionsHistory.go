package whitebit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_positionsHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol     *string
	positionID *int64
	startTime  *int64
	endTime    *int64
	limit      *int64
	offset     *int64
}

// ====== builder-методы ======

func (s *futures_positionsHistory) Symbol(symbol string) *futures_positionsHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_positionsHistory) PositionID(id int64) *futures_positionsHistory {
	s.positionID = &id
	return s
}

func (s *futures_positionsHistory) StartTime(ts int64) *futures_positionsHistory {
	s.startTime = &ts
	return s
}

func (s *futures_positionsHistory) EndTime(ts int64) *futures_positionsHistory {
	s.endTime = &ts
	return s
}

func (s *futures_positionsHistory) Limit(limit int64) *futures_positionsHistory {
	s.limit = &limit
	return s
}

func (s *futures_positionsHistory) Offset(offset int64) *futures_positionsHistory {
	s.offset = &offset
	return s
}

// normalizeTimestampToSeconds — WhiteBIT ожидает Unix time в секундах.
// Если нам передали миллисекунды — конвертим.
func normalizeTimestampToSeconds(ts int64) int64 {
	// всё, что больше ~ 10^12, будем считать миллисекундами
	if ts > 1_000_000_000_000 {
		return ts / 1000
	}
	return ts
}

func (s *futures_positionsHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_PositionsHistory, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/collateral-account/positions/history",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	// НЕ обязательные поля — добавляем только если заданы
	if s.symbol != nil {
		m["market"] = *s.symbol
	}
	if s.positionID != nil {
		m["positionId"] = *s.positionID
	}
	if s.startTime != nil {
		m["startDate"] = normalizeTimestampToSeconds(*s.startTime)
	}
	if s.endTime != nil {
		m["endDate"] = normalizeTimestampToSeconds(*s.endTime)
	}
	if s.limit != nil && *s.limit > 0 {
		m["limit"] = *s.limit
	}
	if s.offset != nil && *s.offset >= 0 {
		m["offset"] = *s.offset
	}

	// Для WhiteBIT приватных v4 эндпоинтов — JSON body (form)
	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ []futures_positionsHistoryWB
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertPositionsHistory(answ), nil
}

// ====== структура под ответ WhiteBIT ======

type futures_positionsHistoryWB struct {
	PositionId       int64   `json:"positionId"`
	Market           string  `json:"market"`
	OpenDate         float64 `json:"openDate"`
	ModifyDate       float64 `json:"modifyDate"`
	Amount           string  `json:"amount"`
	BasePrice        string  `json:"basePrice"`
	RealizedFunding  string  `json:"realizedFunding"`
	LiquidationPrice *string `json:"liquidationPrice"`
	LiquidationState *string `json:"liquidationState"`
	OrderDetail      *struct {
		Id          int64   `json:"id"`
		TradeAmount string  `json:"tradeAmount"`
		Price       string  `json:"price"`
		TradeFee    string  `json:"tradeFee"`
		FundingFee  *string `json:"fundingFee"`
		RealizedPnl *string `json:"realizedPnl"`
	} `json:"orderDetail"`
	PositionSide string `json:"positionSide"`
}
