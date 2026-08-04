package gateio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============GetInstrumentsInfo=================
type futures_getInstrumentsInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol *string
	settle *string
}

func (s *futures_getInstrumentsInfo) Symbol(symbol string) *futures_getInstrumentsInfo {
	s.symbol = &symbol
	return s
}

func (s *futures_getInstrumentsInfo) Settle(settle string) *futures_getInstrumentsInfo {
	s.settle = &settle
	return s
}

func (s *futures_getInstrumentsInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_InstrumentsInfo, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v4/futures/{settle}/contracts",
		SecType:  utils.SecTypeNone,
	}

	settleDefault := "usdt"

	if s.settle == nil {
		s.settle = &settleDefault
	}

	r.Endpoint = strings.Replace(r.Endpoint, "{settle}", *s.settle, 1)

	if s.symbol != nil {
		r.Endpoint = fmt.Sprintf("%s/%s", r.Endpoint, *s.symbol)
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	if s.symbol != nil {
		answ := futures_instrumentsInfo{}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}

		return s.convert.convertInstrumentsInfo([]futures_instrumentsInfo{answ}), nil

	} else {
		answ := []futures_instrumentsInfo{}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}

		return s.convert.convertInstrumentsInfo(answ), nil
	}

}

type futures_instrumentsInfo struct {
	Name              string `json:"name"`
	Mark_price_round  string `json:"mark_price_round"`
	Order_price_round string `json:"order_price_round"`
	Order_size_min    int64  `json:"order_size_min"`
	Quanto_multiplier string `json:"quanto_multiplier"`
	Leverage_max      string `json:"leverage_max"`
	In_delisting      bool   `json:"in_delisting"`
	Status            string `json:"status"`
}
