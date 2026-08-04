package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_setMarginMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	user       string
	symbol     *string
	marginMode *entity.MarginModeType
}

func (s *futures_setMarginMode) Symbol(symbol string) *futures_setMarginMode {
	s.symbol = &symbol
	return s
}

func (s *futures_setMarginMode) MarginMode(marginMode entity.MarginModeType) *futures_setMarginMode {
	s.marginMode = &marginMode
	return s
}

func (s *futures_setMarginMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_MarginMode, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return res, fmt.Errorf("hyperliquid futures setMarginMode: symbol is required")
	}
	if s.marginMode == nil {
		return res, fmt.Errorf("hyperliquid futures setMarginMode: marginMode is required")
	}

	asset, coin, leverage, err := s.getCurrentAssetAndLeverage(ctx, strings.TrimSpace(*s.symbol), opts...)
	if err != nil {
		return res, err
	}

	isCross := true
	switch *s.marginMode {
	case entity.MarginModeTypeCross:
		isCross = true
	case entity.MarginModeTypeIsolated:
		isCross = false
	default:
		return res, fmt.Errorf("hyperliquid futures setMarginMode: unsupported marginMode %q", string(*s.marginMode))
	}

	action := futuresUpdateLeverageAction{
		Type:     "updateLeverage",
		Asset:    asset,
		IsCross:  isCross,
		Leverage: leverage,
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

	_ = coin
	return entity.Futures_MarginMode{MarginMode: string(*s.marginMode)}, nil
}

func (s *futures_setMarginMode) getCurrentAssetAndLeverage(ctx context.Context, symbol string, opts ...utils.RequestOption) (asset int, coin string, leverage int64, err error) {
	if strings.TrimSpace(s.user) == "" {
		return 0, "", 0, fmt.Errorf("hyperliquid futures setMarginMode: main user address is empty")
	}

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
		return 0, "", 0, err
	}

	answ := futures_clearinghouseState{}
	if err := json.Unmarshal(data, &answ); err != nil {
		return 0, "", 0, err
	}

	for idx, item := range answ.AssetPositions {
		coin = strings.TrimSpace(item.Position.Coin)
		if coin == "" {
			continue
		}
		if futuresMatchSymbol(symbol, coin, idx) {
			if item.Position.Leverage.Value <= 0 {
				return 0, "", 0, fmt.Errorf("hyperliquid futures setMarginMode: leverage not found for %s", symbol)
			}
			return idx, coin, item.Position.Leverage.Value, nil
		}
	}

	return 0, "", 0, fmt.Errorf("hyperliquid futures setMarginMode: open position not found for %s", symbol)
}

type futuresUpdateLeverageAction struct {
	Type     string `json:"type"`
	Asset    int    `json:"asset"`
	IsCross  bool   `json:"isCross"`
	Leverage int64  `json:"leverage"`
}
