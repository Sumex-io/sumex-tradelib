package whitebit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// =================== Get Futures Balance (collateral-account) ===================

type futures_getBalance struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts
}

// Если захотим фильтровать по конкретному тикеру — можно позже добавить поле asset
// и метод Asset(...) по аналогии с другими биржами.

// Do запрашивает баланс фьючерсного (collateral) аккаунта.
func (s *futures_getBalance) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.FuturesBalance, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/collateral-account/balance",
		SecType:  utils.SecTypeSigned,
	}

	// У этого эндпоинта параметры не обязательны, можно слать пустой body.
	r.SetFormParams(utils.Params{})

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// Ответ сейчас приходит в виде:
	// {
	//   "USDT": "0",
	//   "BTC": "0",
	//   ...
	// }
	var answ futures_Balance
	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertBalance(answ), nil
}

// map[asset]balance, значение — строка с числом ("0", "12.34" и т.п.)
type futures_Balance map[string]string
