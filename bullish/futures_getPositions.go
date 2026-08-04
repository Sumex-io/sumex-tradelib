package bullish

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
	uid    *string
}

func (s *futures_getPositions) Symbol(symbol string) *futures_getPositions {
	s.symbol = &symbol
	return s
}

func (s *futures_getPositions) UID(uid string) *futures_getPositions {
	s.uid = &uid
	return s
}

func (s *futures_getPositions) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_Positions, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/trading-api/v1/derivatives-positions",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.uid != nil {
		m["tradingAccountId"] = *s.uid
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	answ := []futures_Position{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	res = s.convert.convertPositions(answ)
	return res, nil
}

type futures_Position struct {
	Symbol             string `json:"symbol"`
	Side               string `json:"side"`
	Quantity           string `json:"quantity"`
	Notional           string `json:"notional"`
	EntryNotional      string `json:"entryNotional"`
	MtmPnl             string `json:"mtmPnl"`
	ReportedMtmPnl     string `json:"reportedMtmPnl"`
	ReportedFundingPnl string `json:"reportedFundingPnl"`
	RealizedPnl        string `json:"realizedPnl"`

	CreatedTime string `json:"createdAtTimestamp"`
	UpdatedTime string `json:"updatedAtTimestamp"`
}
