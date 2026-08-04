package binance

import (
	"context"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getMarketCandle struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	isCOINM bool
	convert futures_converts

	timeFrame *entity.TimeFrameType
	symbol    *string
	limit     *int
}

func (s *futures_getMarketCandle) Symbol(symbol string) *futures_getMarketCandle {
	s.symbol = &symbol
	return s
}

func (s *futures_getMarketCandle) TimeFrame(timeFrame entity.TimeFrameType) *futures_getMarketCandle {
	s.timeFrame = &timeFrame
	return s
}

func (s *futures_getMarketCandle) Limit(limit int) *futures_getMarketCandle {
	s.limit = &limit
	return s
}

func (s *futures_getMarketCandle) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_MarketCandle, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/fapi/v1/klines",
		SecType:  utils.SecTypeNone,
	}

	if s.isCOINM {
		r.Endpoint = "/dapi/v1/klines"
	}

	m := utils.Params{}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.timeFrame != nil {
		m["interval"] = strings.ToLower(string(*s.timeFrame))
	}

	if s.limit != nil {
		m["limit"] = *s.limit
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ, err := utils.NewJSON(data)
	if err != nil {
		return res, err
	}

	res = s.convert.convertMarketCandle(answ)
	return res, nil
}

type futures_marketCandle_responce struct {
	Open               string `json:"open"`
	High               string `json:"high"`
	Low                string `json:"low"`
	Close              string `json:"close"`
	Volume             string `json:"volume"`
	CreatedAtTimestamp string `json:"createdAtTimestamp"`
}
