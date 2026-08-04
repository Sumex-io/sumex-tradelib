package okx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============GetPositions=================
type futures_getPositions struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts
}

func (s *futures_getPositions) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_Positions, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v5/account/positions",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []futures_Position `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	return s.convert.convertPositions(answ.Result), nil
}

type futures_Position struct {
	InstID      string `json:"instId"`
	PosCcy      string `json:"posCcy,omitempty"`
	LiabCcy     string `json:"liabCcy,omitempty"`
	OptVal      string `json:"optVal,omitempty"`
	Ccy         string `json:"ccy"`
	PosID       string `json:"posId"`
	TradeID     string `json:"tradeId"`
	Pos         string `json:"pos"`
	AvailPos    string `json:"availPos,omitempty"`
	AvgPx       string `json:"avgPx"`
	Upl         string `json:"upl"`
	RealizedPnl string `json:"realizedPnl"`
	UplRatio    string `json:"uplRatio"`
	Lever       string `json:"lever"`
	LiqPx       string `json:"liqPx,omitempty"`
	MarkPx      string `json:"markPx,omitempty"`
	Imr         string `json:"imr,omitempty"`
	Margin      string `json:"margin,omitempty"`
	MgnRatio    string `json:"mgnRatio"`
	Mmr         string `json:"mmr"`
	Liab        string `json:"liab,omitempty"`
	Interest    string `json:"interest"`
	NotionalUsd string `json:"notionalUsd"`
	ADL         string `json:"adl"`
	Last        string `json:"last"`
	DeltaBS     string `json:"deltaBS"`
	DeltaPA     string `json:"deltaPA"`
	GammaBS     string `json:"gammaBS"`
	GammaPA     string `json:"gammaPA"`
	ThetaBS     string `json:"thetaBS"`
	ThetaPA     string `json:"thetaPA"`
	VegaBS      string `json:"vegaBS"`
	VegaPA      string `json:"vegaPA"`
	PosSide     string `json:"posSide"`
	MgnMode     string `json:"mgnMode"`
	InstType    string `json:"instType"`
	CTime       string `json:"cTime"`
	UTime       string `json:"uTime"`
}
