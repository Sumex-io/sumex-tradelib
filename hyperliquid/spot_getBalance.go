package hyperliquid

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getBalance struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts
}

func (s *spot_getBalance) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.AssetsBalance, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/info",
		SecType:  utils.SecTypeNone,
	}

	b, _ := json.Marshal(map[string]interface{}{
		"type": "spotClearinghouseState",
		"user": "",
	})
	r.BodyString = string(b)

	opts = append(opts, func(rr *utils.Request) error {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(rr.BodyString), &m); err != nil {
			return err
		}
		if u, ok := m["user"].(string); !ok || u == "" {
			m["user"] = normalizeHexAddress(rr.TmpApi)
			nb, _ := json.Marshal(m)
			rr.BodyString = string(nb)
		}
		return nil
	})

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := spot_spotClearinghouseState{}
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertSpotBalances(answ), nil
}

type spot_spotClearinghouseState struct {
	Balances []struct {
		Coin     string `json:"coin"`
		Token    int64  `json:"token"`
		Hold     string `json:"hold"`
		Total    string `json:"total"`
		EntryNtl string `json:"entryNtl"`
	} `json:"balances"`
}
