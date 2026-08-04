package binance

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
		Endpoint: "/api/v3/exchangeInfo",
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
	answ := spot_instrumentsInfo{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertInstrumentsInfo(answ), nil
}

type spot_instrumentsInfo struct {
	ServerTime int64 `json:"serverTime"`
	Symbols    []struct {
		Symbol             string        `json:"symbol"`
		Status             string        `json:"status"`
		BaseAsset          string        `json:"baseAsset"`
		QuoteAsset         string        `json:"quoteAsset"`
		BaseAssetPrecision int64         `json:"baseAssetPrecision"`
		Filters            []interface{} `json:"filters"`
	} `json:"symbols"`
}
