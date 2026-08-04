package bitget

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getPositions struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol *string
}

func (s *futures_getPositions) Symbol(symbol string) *futures_getPositions {
	s.symbol = &symbol
	return s
}

func (s *futures_getPositions) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_Positions, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v2/mix/position/all-position",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"productType": "USDT-FUTURES", "marginCoin": "USDT"}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	r.SetParams(m)

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
	res = s.convert.convertPositions(answ.Result)
	return res, nil
}

type futures_Position struct {
	Symbol          string `json:"symbol"`
	HoldSide        string `json:"holdSide"`
	Total           string `json:"total"`
	Available       string `json:"available"`
	Leverage        string `json:"leverage"`
	AchievedProfits string `json:"achievedProfits"`
	OpenPriceAvg    string `json:"openPriceAvg"`
	MarginMode      string `json:"marginMode"`
	PosMode         string `json:"posMode"`
	UnrealizedPL    string `json:"unrealizedPL"`
	MarkPrice       string `json:"markPrice"`
	TotalFee        string `json:"totalFee"`
	DeductedFee     string `json:"deductedFee"`

	CreateTime string `json:"cTime"`
	UpdateTime string `json:"uTime"`
}
