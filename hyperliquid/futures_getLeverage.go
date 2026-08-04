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

type futures_getLeverage struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	user       string
	symbol     *string
	marginMode *entity.MarginModeType
}

func (s *futures_getLeverage) Symbol(symbol string) *futures_getLeverage {
	s.symbol = &symbol
	return s
}

func (s *futures_getLeverage) MarginMode(marginMode entity.MarginModeType) *futures_getLeverage {
	s.marginMode = &marginMode
	return s
}

func (s *futures_getLeverage) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_Leverage, err error) {
	if strings.TrimSpace(s.user) == "" {
		return res, fmt.Errorf("hyperliquid futures getLeverage: main user address is empty")
	}
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return res, fmt.Errorf("hyperliquid futures getLeverage: symbol is required")
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
		return res, err
	}
	// log.Println("=b8e07d=", string(data))
	answ := futures_clearinghouseState{}
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertLeverage(answ, strings.TrimSpace(*s.symbol), s.marginMode), nil
}

func futuresMatchSymbol(input string, coin string, assetID int) bool {
	want := strings.ToUpper(strings.TrimSpace(input))
	if want == "" {
		return false
	}

	coin = strings.ToUpper(strings.TrimSpace(coin))
	if want == coin || want == coin+"/USDC" || want == strconv.Itoa(assetID) {
		return true
	}

	wantNoSlash := strings.ReplaceAll(want, "/", "")
	coinNoSlash := coin + "USDC"
	return wantNoSlash == coinNoSlash
}
