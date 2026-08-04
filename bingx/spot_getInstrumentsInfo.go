package bingx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getInstrumentsInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts
	symbol  *string
}

func (s *spot_getInstrumentsInfo) Symbol(symbol string) *spot_getInstrumentsInfo {
	s.symbol = &symbol
	return s
}

func (s *spot_getInstrumentsInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_InstrumentsInfo, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/openApi/spot/v1/common/symbols",
		SecType:  utils.SecTypeNone,
	}

	m := utils.Params{}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result spot_instrumentsInfo `json:"data"`
	}
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	res = s.convert.convertInstrumentsInfo(answ.Result)
	return res, nil
}

type spot_instrumentsInfo struct {
	Symbols []struct {
		Symbol      string  `json:"symbol"`
		MinQty      float64 `json:"minQty"`
		MaxQty      float64 `json:"maxQty"`
		MinNotional float64 `json:"minNotional"`
		MaxNotional float64 `json:"maxNotional"`
		Status      int64   `json:"status"`
		TickSize    float64 `json:"tickSize"`
		StepSize    float64 `json:"stepSize"`
	} `json:"symbols"`
}
