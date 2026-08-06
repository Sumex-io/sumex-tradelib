package katana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_cancelOrder struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       futures_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)
	sign          *katanaSigner

	symbol  *string
	orderID *string
}

// Symbol is accepted for parity with every other connector's cancelOrder. Katana's DELETE
// /v1/orders is scoped by orderIds alone and the docs give the id precedence over every other
// filter, so sending a market alongside it could only ever contradict the id.
func (s *futures_cancelOrder) Symbol(symbol string) *futures_cancelOrder {
	s.symbol = &symbol
	return s
}

func (s *futures_cancelOrder) OrderID(orderID string) *futures_cancelOrder {
	s.orderID = &orderID
	return s
}

// katanaCancelByOrderIdsParams is the DELETE /v1/orders `parameters` object scoped by orderIds.
type katanaCancelByOrderIdsParams struct {
	Nonce        string   `json:"nonce"`
	Wallet       string   `json:"wallet"`
	DelegatedKey string   `json:"delegatedKey,omitempty"`
	OrderIds     []string `json:"orderIds"`
}

// Do sends DELETE /v1/orders scoped to a single orderId. Ts is the client-observed time: the
// response carries only orderId/clientOrderId/status, so no exchange timestamp exists to report.
//
// Checking `status` is load-bearing, not defensive: the docs require the client to confirm
// cancellation from the response, and the only values are canceled/notFound. Treating notFound as
// success would let amendOrder — which cancels through here — place a fresh resting order on top of
// a position whose original order had just filled.
func (s *futures_cancelOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	if s.orderID == nil || strings.TrimSpace(*s.orderID) == "" {
		return nil, errors.New("katana: cancelOrder requires an orderID")
	}
	orderID := strings.TrimSpace(*s.orderID)

	wallet, err := s.resolveWallet(ctx, opts...)
	if err != nil {
		return nil, err
	}
	delegated, err := s.sign.delegatedAddress()
	if err != nil {
		return nil, err
	}
	nonceStr, nonceBig, err := newNonce()
	if err != nil {
		return nil, err
	}

	sig, err := s.sign.signCancelByOrderIds(nonceBig, wallet, []string{orderID})
	if err != nil {
		return nil, err
	}

	body := katanaSignedRequest[katanaCancelByOrderIdsParams]{
		Parameters: katanaCancelByOrderIdsParams{
			Nonce:        nonceStr,
			Wallet:       wallet,
			DelegatedKey: delegated,
			OrderIds:     []string{orderID},
		},
		Signature: sig,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	r := &utils.Request{
		Method:   http.MethodDelete,
		Endpoint: "/v1/orders",
		SecType:  utils.SecTypeSigned,
	}
	r.BodyString = string(b)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}

	var answ []katanaCancelResult
	if err := json.Unmarshal(data, &answ); err != nil {
		return nil, err
	}
	if len(answ) == 0 {
		return nil, fmt.Errorf("katana: cancel response for order %s was empty", orderID)
	}

	for _, item := range answ {
		if !strings.EqualFold(item.Status, "canceled") {
			return nil, fmt.Errorf("katana: order %s was not confirmed canceled (status %q)", firstNonEmpty(item.OrderId, orderID), item.Status)
		}
	}

	return s.convert.convertCancelOrder(answ, time.Now().UnixMilli()), nil
}

// firstNonEmpty lets cancelOrder name an order in an error even when Katana's response omits
// `orderId`, which it does for a `notFound` order specified by clientOrderId.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
