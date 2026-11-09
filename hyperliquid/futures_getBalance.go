package hyperliquid

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_getBalance struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts
	user    string
}

func (s *futures_getBalance) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.FuturesBalance, err error) {
	answ := futures_clearinghouseState{}
	if err := s.info(ctx, "clearinghouseState", &answ, opts...); err != nil {
		return res, err
	}

	var mode string
	if err := s.info(ctx, "userAbstraction", &mode, opts...); err != nil {
		return res, err
	}
	if mode != "unifiedAccount" && mode != "portfolioMargin" {
		return s.convert.convertBalance(answ), nil
	}

	// Unified and portfolio-margin accounts keep collateral in the spot clearinghouse; perps withdrawable reads 0.
	spot := spot_spotClearinghouseState{}
	if err := s.info(ctx, "spotClearinghouseState", &spot, opts...); err != nil {
		return res, err
	}
	return s.convert.convertUnifiedBalance(answ, spot), nil
}

func (s *futures_getBalance) info(ctx context.Context, infoType string, out interface{}, opts ...utils.RequestOption) error {
	b, _ := json.Marshal(map[string]interface{}{
		"type": infoType,
		"user": s.user,
	})
	r := &utils.Request{
		Method:     http.MethodPost,
		Endpoint:   "/info",
		SecType:    utils.SecTypeNone,
		BodyString: string(b),
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
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
