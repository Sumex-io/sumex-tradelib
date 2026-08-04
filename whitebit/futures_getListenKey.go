package whitebit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// =================== GetListenKey (websocket_token) ===================

type futures_getListenKey struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts
}

// Do запрашивает websocket_token для приватного WS
func (s *futures_getListenKey) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_ListenKey, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/profile/websocket_token",
		SecType:  utils.SecTypeSigned,
	}

	// Для whitebit тело {request, nonce} формируется внутри sign-builder’а,
	// поэтому здесь ничего не добавляем.
	// r.Form оставляем пустым.

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ futures_listenKey
	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertListenKey(answ), nil
}

// структура под ответ whitebit
type futures_listenKey struct {
	WebsocketToken string `json:"websocket_token"`
}
