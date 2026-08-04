package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_setLeverage struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol     *string
	leverage   *string
	marginMode *entity.MarginModeType
}

func (s *futures_setLeverage) Symbol(symbol string) *futures_setLeverage {
	s.symbol = &symbol
	return s
}

func (s *futures_setLeverage) Leverage(leverage string) *futures_setLeverage {
	s.leverage = &leverage
	return s
}

func (s *futures_setLeverage) MarginMode(marginMode entity.MarginModeType) *futures_setLeverage {
	s.marginMode = &marginMode
	return s
}

func (s *futures_setLeverage) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_Leverage, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return res, fmt.Errorf("hyperliquid futures setLeverage: symbol is required")
	}
	if s.leverage == nil || strings.TrimSpace(*s.leverage) == "" {
		return res, fmt.Errorf("hyperliquid futures setLeverage: leverage is required")
	}

	asset, coin, err := resolvePerpAsset(ctx, s.callAPI, strings.TrimSpace(*s.symbol), opts...)
	if err != nil {
		return res, err
	}

	lev, err := strconv.ParseInt(strings.TrimSpace(*s.leverage), 10, 64)
	if err != nil || lev <= 0 {
		return res, fmt.Errorf("hyperliquid futures setLeverage: invalid leverage %q", *s.leverage)
	}

	isCross := true
	if s.marginMode != nil {
		switch *s.marginMode {
		case entity.MarginModeTypeCross:
			isCross = true
		case entity.MarginModeTypeIsolated:
			isCross = false
		default:
			return res, fmt.Errorf("hyperliquid futures setLeverage: unsupported marginMode %q", string(*s.marginMode))
		}
	}

	action := futuresSetLeverageAction{
		Type:     "updateLeverage",
		Asset:    asset,
		IsCross:  isCross,
		Leverage: lev,
	}

	b, _ := json.Marshal(action)

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/exchange",
		SecType:  utils.SecTypeSigned,
	}
	r.BodyString = string(b)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Status   string `json:"status"`
		Response struct {
			Type string `json:"type"`
		} `json:"response"`
	}
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertSetLeverage(coin+"/USDC", lev), nil
}

type futuresSetLeverageAction struct {
	Type     string `json:"type"`
	Asset    int    `json:"asset"`
	IsCross  bool   `json:"isCross"`
	Leverage int64  `json:"leverage"`
}
