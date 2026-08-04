package hyperliquid

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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
		Method:   http.MethodPost,
		Endpoint: "/info",
		SecType:  utils.SecTypeNone,
	}

	b, _ := json.Marshal(map[string]interface{}{
		"type": "meta",
	})
	r.BodyString = string(b)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := perpMetaResponse{}
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	res = s.convert.convertInstrumentsInfo(answ)

	if s.symbol != nil && strings.TrimSpace(*s.symbol) != "" {
		want := strings.ToUpper(strings.TrimSpace(*s.symbol))
		filtered := make([]entity.Futures_InstrumentsInfo, 0, len(res))

		for _, it := range res {
			if futuresInstrumentMatches(it, want) {
				filtered = append(filtered, it)
			}
		}
		return filtered, nil
	}

	return res, nil
}

func futuresInstrumentMatches(it entity.Futures_InstrumentsInfo, want string) bool {
	symbol := strings.ToUpper(strings.TrimSpace(it.Symbol))
	base := strings.ToUpper(strings.TrimSpace(it.Base))
	tokenID := strings.TrimSpace(it.TokenId)

	if symbol == want || base == want || tokenID == want {
		return true
	}

	wantNoSlash := strings.ReplaceAll(want, "/", "")
	symbolNoSlash := strings.ReplaceAll(symbol, "/", "")
	if wantNoSlash != "" && symbolNoSlash == wantNoSlash {
		return true
	}

	if strings.HasSuffix(want, "/USDC") && base == strings.TrimSuffix(want, "/USDC") {
		return true
	}

	if !strings.Contains(want, "/") && strings.HasSuffix(want, "USDC") {
		baseFromWant := strings.TrimSuffix(want, "USDC")
		if baseFromWant != "" && base == baseFromWant {
			return true
		}
	}

	if n, err := strconv.Atoi(want); err == nil && n >= 0 {
		return tokenID == strconv.Itoa(n)
	}

	return false
}
