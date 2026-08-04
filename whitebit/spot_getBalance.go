package whitebit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============GetBalance (SPOT)=================

type spot_getBalance struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
}

// внутренний формат ответа WhiteBIT spot trade-account/balance
type spot_balanceItem struct {
	Available string `json:"available"`
	Freeze    string `json:"freeze"`
}

func (s *spot_getBalance) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.AssetsBalance, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/trade-account/balance",
		SecType:  utils.SecTypeSigned,
	}

	// тело для WhiteBIT формируется внутри callAPI (nonce + request)
	// здесь дополнительных параметров нет
	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// ожидаем объект вида:
	// {
	//   "BTC": { "available": "0.5", "freeze": "0.1" },
	//   "USDT": { "available": "100", "freeze": "0" }
	// }
	raw := map[string]spot_balanceItem{}

	if err = json.Unmarshal(data, &raw); err != nil {
		return res, err
	}

	for asset, item := range raw {
		// фильтруем нулевые балансы (и свободный, и замороженный 0)
		if item.Available == "0" && item.Freeze == "0" {
			continue
		}

		res = append(res, entity.AssetsBalance{
			Asset:   asset,
			Balance: item.Available, // свободный баланс
			Locked:  item.Freeze,    // замороженный (ордеры и т.п.)
		})
	}

	return res, nil
}
