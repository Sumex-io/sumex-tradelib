package hyperliquid

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getBalance struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts
	user    string
}

func (s *futures_getBalance) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.FuturesBalance, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/info",
		SecType:  utils.SecTypeNone,
	}

	b, _ := json.Marshal(map[string]interface{}{
		"type": "clearinghouseState",
		"user": s.user,
	})
	r.BodyString = string(b)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := futures_clearinghouseState{}
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertBalance(answ), nil
}

type futures_clearinghouseState struct {
	MarginSummary struct {
		AccountValue    string `json:"accountValue"`
		TotalMarginUsed string `json:"totalMarginUsed"`
	} `json:"marginSummary"`
	CrossMarginSummary struct {
		AccountValue    string `json:"accountValue"`
		TotalMarginUsed string `json:"totalMarginUsed"`
	} `json:"crossMarginSummary"`
	Withdrawable   string `json:"withdrawable"`
	AssetPositions []struct {
		Type     string `json:"type"`
		Position struct {
			Coin           string `json:"coin"`
			Szi            string `json:"szi"`
			EntryPx        string `json:"entryPx"`
			PositionValue  string `json:"positionValue"`
			UnrealizedPnl  string `json:"unrealizedPnl"`
			ReturnOnEquity string `json:"returnOnEquity"`
			Leverage       struct {
				Type  string `json:"type"`
				Value int64  `json:"value"`
			} `json:"leverage"`
			MarginUsed  string `json:"marginUsed"`
			MaxLeverage int64  `json:"maxLeverage"`
		} `json:"position"`
	} `json:"assetPositions"`
	Time int64 `json:"time"`
}
