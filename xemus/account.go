package xemus

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

// ===================GetAccountInfo==================

type getAccountInfo struct {
	callAPI callAPIFunc
}

// Do proves the key pair works (GET /v1/account/info is the cheapest signed read) and reports what
// it may do. perp-api issues every user key with the fixed scope read,trading and never
// withdrawal, so the permissions are known rather than guessed. The Orderly account id is the uid.
func (s *getAccountInfo) Do(ctx context.Context) (res entity.AccountInformation, err error) {
	data, err := s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/account/info", signed: true})
	if err != nil {
		return res, err
	}
	var answ accountInfoResponse
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}
	return entity.AccountInformation{
		UID:         answ.AccountID,
		IP:          "0.0.0.0/0",
		CanRead:     true,
		CanTrade:    true,
		CanTransfer: false,
		PermSpot:    false,
		PermFutures: true,
	}, nil
}

// ===================SignAuthStream==================

type signAuthStream struct {
	callAPI callAPIFunc

	timeStamp *int64
}

// TimeStamp is accepted for parity with the other connectors and unused: perp-api signs Orderly's
// private-stream login with its own timestamp, which is the one Orderly checks.
func (s *signAuthStream) TimeStamp(timeStamp int64) *signAuthStream {
	s.timeStamp = &timeStamp
	return s
}

// Do asks perp-api for the private WebSocket login (POST /v1/account/stream-signature). The client
// connects to Orderly directly and needs all four values Orderly's auth message carries — the
// account id for the URL, the Orderly key, and the timestamp the signature covers — so they travel
// alongside the signature rather than being dropped.
func (s *signAuthStream) Do(ctx context.Context) (res entity.SignAuthStream, err error) {
	data, err := s.callAPI(ctx, &request{method: http.MethodPost, path: "/v1/account/stream-signature", signed: true})
	if err != nil {
		return res, err
	}
	var answ streamSignatureResponse
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}
	return entity.SignAuthStream{
		Signature: answ.Signature,
		Timestamp: answ.Timestamp.int64(),
		AccountID: answ.AccountID,
		Key:       answ.OrderlyKey,
		URL:       answ.URL,
	}, nil
}
