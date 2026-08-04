package okx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getInstrumentsInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol *string
}

func (s *futures_getInstrumentsInfo) Symbol(symbol string) *futures_getInstrumentsInfo {
	s.symbol = &symbol
	return s
}

func (s *futures_getInstrumentsInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_InstrumentsInfo, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v5/public/instruments",
		SecType:  utils.SecTypeNone,
	}

	m := utils.Params{
		"instType": "SWAP",
	}

	if s.symbol != nil {
		m["instId"] = *s.symbol
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result []futures_instrumentsInfo `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	return s.convert.convertInstrumentsInfo(answ.Result), nil
}

type futures_instrumentsInfo struct {
	InstId    string `json:"instId"`
	CtMult    string `json:"ctMult"`
	CtVal     string `json:"ctVal"`
	CtValCcy  string `json:"ctValCcy"`
	SettleCcy string `json:"settleCcy"`
	Lever     string `json:"lever"`
	LotSz     string `json:"lotSz"`
	MinSz     string `json:"minSz"`
	State     string `json:"state"`
	TickSz    string `json:"tickSz"`
}
