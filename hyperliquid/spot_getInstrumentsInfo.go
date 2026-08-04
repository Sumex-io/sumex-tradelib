package hyperliquid

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getInstrumentsInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts
	symbol  *string
}

func (s *spot_getInstrumentsInfo) Symbol(symbol string) *spot_getInstrumentsInfo {
	s.symbol = &symbol
	return s
}

func (s *spot_getInstrumentsInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_InstrumentsInfo, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/info",
		SecType:  utils.SecTypeNone,
	}

	b, _ := json.Marshal(map[string]interface{}{
		"type": "spotMeta",
	})
	r.BodyString = string(b)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := spotMetaResponse{}
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	res = s.convert.convertInstrumentsInfo(answ)

	if s.symbol != nil && strings.TrimSpace(*s.symbol) != "" {
		want := strings.ToUpper(strings.TrimSpace(*s.symbol))
		filtered := make([]entity.Spot_InstrumentsInfo, 0, len(res))
		for _, it := range res {
			if strings.ToUpper(strings.TrimSpace(it.Symbol)) == want || strings.ToUpper(strings.TrimSpace(it.Base)) == want {
				filtered = append(filtered, it)
			}
		}
		return filtered, nil
	}

	return res, nil
}
