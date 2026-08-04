package bullish

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getMarketCandle struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
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
		Endpoint: "/trading-api/v1/markets/{symbol}/candle",
		SecType:  utils.SecTypeNone,
	}

	start := time.Now().UTC()
	end := start.Add(-2400 * time.Hour)

	m := utils.Params{"_pageSize": 100, "createdAtDatetime[lte]": start.Format("2006-01-02T15:04:05") + ".000Z", "createdAtDatetime[gte]": end.Format("2006-01-02T15:04:05") + ".000Z"}
	if s.symbol != nil {
		r.Endpoint = strings.Replace(r.Endpoint, "{symbol}", *s.symbol, 1)
	}

	if s.timeFrame != nil {
		m["timeBucket"] = strings.ToLower(string(*s.timeFrame))
	}

	if s.limit != nil {
		m["_pageSize"] = *s.limit
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	answ := []futures_marketCandle_responce{}

	err = json.Unmarshal(data, &answ)
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
