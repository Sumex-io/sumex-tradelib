package weex

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
}

func (s *futures_getPositions) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_Positions, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/capi/v3/account/position/allPosition",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ []futures_Position
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertPositions(answ), nil
}

type futures_Position struct {
	ID                         int64  `json:"id"`
	Asset                      string `json:"asset"`
	Symbol                     string `json:"symbol"`
	Side                       string `json:"side"`
	MarginType                 string `json:"marginType"`
	SeparatedMode              string `json:"separatedMode"`
	SeparatedOpenOrderID       int64  `json:"separatedOpenOrderId"`
	Leverage                   string `json:"leverage"`
	Size                       string `json:"size"`
	OpenValue                  string `json:"openValue"`
	OpenFee                    string `json:"openFee"`
	FundingFee                 string `json:"fundingFee"`
	MarginSize                 string `json:"marginSize"`
	IsolatedMargin             string `json:"isolatedMargin"`
	IsAutoAppendIsolatedMargin bool   `json:"isAutoAppendIsolatedMargin"`
	CumOpenSize                string `json:"cumOpenSize"`
	CumOpenValue               string `json:"cumOpenValue"`
	CumOpenFee                 string `json:"cumOpenFee"`
	CumCloseSize               string `json:"cumCloseSize"`
	CumCloseValue              string `json:"cumCloseValue"`
	CumCloseFee                string `json:"cumCloseFee"`
	CumFundingFee              string `json:"cumFundingFee"`
	CumLiquidateFee            string `json:"cumLiquidateFee"`
	CreatedMatchSequenceID     int64  `json:"createdMatchSequenceId"`
	UpdatedMatchSequenceID     int64  `json:"updatedMatchSequenceId"`
	CreatedTime                int64  `json:"createdTime"`
	UpdatedTime                int64  `json:"updatedTime"`
	UnrealizePnl               string `json:"unrealizePnl"`
	LiquidatePrice             string `json:"liquidatePrice"`
}
