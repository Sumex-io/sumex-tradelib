package katana

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/utils"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/google/uuid"
)

// aPIError carries a non-2xx Katana response. The message is bounded on purpose: Katana's own
// error bodies are a small {"code","message"} object, but everything else that can occupy that
// channel (a 5xx HTML page, a proxy dump) reaches both the caller and any on-disk log the caller
// keeps.
type aPIError struct {
	StatusCode int
	Raw        []byte
}

func (e aPIError) Error() string {
	if e.StatusCode == http.StatusTooManyRequests {
		return "katana rate limit exceeded"
	}
	return fmt.Sprintf("katana: HTTP %d: %s", e.StatusCode, summarizeErrorBody(e.Raw))
}

// maxErrorBodyBytes bounds how much of a non-2xx body reaches the error message.
const maxErrorBodyBytes = 512

// summarizeErrorBody renders a non-2xx body for an error message: whitespace collapsed to one
// line, truncated with an explicit marker so a partial body can never read as a complete one.
func summarizeErrorBody(raw []byte) string {
	body := strings.Join(strings.Fields(string(raw)), " ")
	if body == "" {
		return "(empty response body)"
	}
	if len(body) > maxErrorBodyBytes {
		return body[:maxErrorBodyBytes] + "... (truncated)"
	}
	return body
}

// ===============SPOT=================

func (c *SpotClient) callAPI(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error) {
	if c.Proxy != "" {
		r.Proxy = c.Proxy
	}
	r.BaseURL = c.BaseURL
	r.TimeOffset = c.TimeOffset
	r.TmpApi = c.apiKey
	r.TmpSig = c.secretKey

	opts = append(opts, createFullURL, createBody, createSign, createHeaders)
	if err := r.ParseRequest(opts...); err != nil {
		return []byte{}, &http.Header{}, err
	}

	// Deliberately narrower than the other connectors in this library: the request headers and
	// the raw *http.Request are NOT dumped, because both carry KP-API-KEY — a durable credential
	// — and KP-HMAC-SIGNATURE.
	c.debug("FullURL %s\n", r.FullURL)
	c.debug("Body %s\n", r.BodyString)

	req, err := http.NewRequest(r.Method, r.FullURL, r.Body)
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	req = req.WithContext(ctx)
	req.Header = r.Header

	res, err := r.DoFunc(req)
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	data, err = r.ReadAllBody(res.Body, res.Header.Get("Content-Encoding"))
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	defer func() {
		cerr := res.Body.Close()
		if err == nil && cerr != nil {
			err = cerr
		}
	}()

	c.debug("response status code: %d\n", res.StatusCode)
	c.debug("response body: %s\n", string(data))

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, &res.Header, aPIError{StatusCode: res.StatusCode, Raw: data}
	}

	return data, &res.Header, nil
}

// ===============FUTURES=================

func (c *FuturesClient) callAPI(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error) {
	if c.Proxy != "" {
		r.Proxy = c.Proxy
	}
	r.BaseURL = c.BaseURL
	r.TimeOffset = c.TimeOffset
	r.TmpApi = c.apiKey
	r.TmpSig = c.secretKey

	// A public read carries no signature, so createHeaders leaves it bare. The key is still
	// worth sending: unkeyed reads share one bucket per caller IP, a keyed one gets the API
	// account its own. Katana validates the key even here -- a wrong one answers 401
	// INVALID_API_KEY -- so only a caller that knows the key belongs to this host sets it.
	if r.SecType == utils.SecTypeNone && c.PublicAPIKey != "" {
		header := http.Header{}
		if r.Header != nil {
			header = r.Header.Clone()
		}
		header.Set("KP-API-KEY", c.PublicAPIKey)
		r.Header = header
	}

	opts = append(opts, createFullURL, createBody, createSign, createHeaders)
	if err := r.ParseRequest(opts...); err != nil {
		return []byte{}, &http.Header{}, err
	}

	c.debug("FullURL %s\n", r.FullURL)
	c.debug("Body %s\n", r.BodyString)

	req, err := http.NewRequest(r.Method, r.FullURL, r.Body)
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	req = req.WithContext(ctx)
	req.Header = r.Header

	res, err := r.DoFunc(req)
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	data, err = r.ReadAllBody(res.Body, res.Header.Get("Content-Encoding"))
	if err != nil {
		return []byte{}, &http.Header{}, err
	}
	defer func() {
		cerr := res.Body.Close()
		if err == nil && cerr != nil {
			err = cerr
		}
	}()

	c.debug("response status code: %d\n", res.StatusCode)
	c.debug("response body: %s\n", string(data))

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, &res.Header, aPIError{StatusCode: res.StatusCode, Raw: data}
	}

	return data, &res.Header, nil
}

// =============END================

// createFullURL appends the query string only when there is one: a bare trailing "?" changes the
// URL but never the HMAC, which is signed over the query string either way.
func createFullURL(r *utils.Request) error {
	fullURL := fmt.Sprintf("%s%s", r.BaseURL, r.Endpoint)
	queryString := r.Query.Encode()
	if queryString != "" {
		fullURL = fmt.Sprintf("%s?%s", fullURL, queryString)
	}
	r.QueryString = queryString
	r.FullURL = fullURL
	return nil
}

// createBody takes the JSON the action already built. Katana bodies are a
// {"parameters":{...},"signature":"0x..."} envelope whose signature is produced by the action
// itself, so nothing is assembled here.
func createBody(r *utils.Request) error {
	if r.BodyString != "" {
		r.Body = bytes.NewBufferString(r.BodyString)
		return nil
	}
	r.Body = &bytes.Buffer{}
	return nil
}

// createSign is the HMAC-SHA256 over the exact bytes that go on the wire: the query string for a
// GET, the raw body for a POST or DELETE. The EIP-712 wallet signature is NOT produced here — see
// the note on katanaSigner.
func createSign(r *utils.Request) error {
	if r.SecType != utils.SecTypeSigned {
		r.TmpSig = ""
		return nil
	}

	sf, err := utils.SignFunc(utils.KeyTypeHmac)
	if err != nil {
		return err
	}

	payload := r.QueryString
	if r.Method == http.MethodPost || r.Method == http.MethodDelete {
		payload = r.BodyString
	}

	sign, err := sf(r.TmpSig, payload)
	if err != nil {
		return err
	}
	r.TmpSig = ""
	r.Sign = *sign
	return nil
}

func createHeaders(r *utils.Request) error {
	header := http.Header{}
	if r.Header != nil {
		header = r.Header.Clone()
	}

	if r.BodyString != "" {
		header.Set("Content-Type", "application/json")
	}
	if r.SecType == utils.SecTypeSigned {
		header.Set("KP-API-KEY", r.TmpApi)
		header.Set("KP-HMAC-SIGNATURE", r.Sign)
	}

	r.TmpApi = ""
	r.Header = header
	return nil
}

// newNonce returns a UUID v1 both as its canonical string (for the request query or body) and as
// the big-endian uint128 the EIP-712 payload expects. Version 1 specifically: Katana reads the
// timestamp it encodes to enforce request timing and reject replays, so a nonce must be minted per
// request and never reused (API_NOTES.md, "Nonce mechanics").
func newNonce() (string, *big.Int, error) {
	u, err := uuid.NewUUID()
	if err != nil {
		return "", nil, err
	}
	return u.String(), new(big.Int).SetBytes(u[:]), nil
}

// ===============EIP-712 SIGNING=================
//
// katanaSigner produces the wallet signature every trade action carries alongside the HMAC. It is
// separate from createSign above, and injected into the action builders rather than run as a
// RequestOption, for one reason: the REST body and the signed struct encode type/side/timeInForce/
// selfTradePrevention/triggerType DIFFERENTLY — POST /v1/orders takes lowercase strings ("market",
// "buy", "gtc", "dc"), the signed `Order` struct takes the numeric enum as a uint8 (API_NOTES.md
// section B). Signing from inside the transport would mean re-deriving the numeric form from the
// string already on the wire, i.e. a second derivation path that could describe a different order
// than the one being sent. The action derives both encodings once and signs before handing the
// finished body to the transport.

// EIP-712 domain separator values, per API_NOTES.md Section B. Do NOT "correct" prodDomainVersion
// to the "1.0.0" that appears in the docs' own code sample: Katana engineering confirmed 2.0.0 and
// that the sample is a documentation bug they are fixing at the source. A wrong version here makes
// every order, cancel and margin change fail with INVALID_WALLET_SIGNATURE.
const (
	prodDomainVersion    = "2.0.0"
	sandboxDomainVersion = "2.0.0-sandbox"
)

const (
	prodChainID    = 747474
	sandboxChainID = 737373

	prodVerifyingContract    = "0x62230CeA619F734cc215bB8074bbF07bE4Eb633e"
	sandboxVerifyingContract = "0x92d3072dDe1aD3e9B7895500F504aA5e664E71d3"
)

type katanaSigner struct {
	privKeyHex string
	demo       bool
}

// domain switches the EIP-712 domain separator on the same demo flag that picks the base URL, so
// signing for one environment and sending to the other is not expressible.
func (s *katanaSigner) domain() apitypes.TypedDataDomain {
	if s.demo {
		return apitypes.TypedDataDomain{
			Name:              "KatanaPerps",
			Version:           sandboxDomainVersion,
			ChainId:           math.NewHexOrDecimal256(sandboxChainID),
			VerifyingContract: sandboxVerifyingContract,
		}
	}
	return apitypes.TypedDataDomain{
		Name:              "KatanaPerps",
		Version:           prodDomainVersion,
		ChainId:           math.NewHexOrDecimal256(prodChainID),
		VerifyingContract: prodVerifyingContract,
	}
}

var domainType = []apitypes.Type{
	{Name: "name", Type: "string"},
	{Name: "version", Type: "string"},
	{Name: "chainId", Type: "uint256"},
	{Name: "verifyingContract", Type: "address"},
}

// orderSignPayload carries the caller-supplied fields of the EIP-712 `Order` struct signed for
// POST /v1/orders (API_NOTES.md Section B, type `Order`). The struct's three always-constant
// fields are not exposed here; orderTypedData fills them in.
//
// DelegatedPublicKey may be left empty to have orderTypedData derive it. If set, it must equal
// that derived address — a mismatch is an error rather than an override, so a caller cannot sign
// one delegated key while sending another on the wire.
type orderSignPayload struct {
	Nonce               *big.Int
	Wallet              string
	MarketSymbol        string
	OrderType           uint8
	OrderSide           uint8
	Quantity            string
	LimitPrice          string
	TriggerPrice        string
	TriggerType         uint8
	IsReduceOnly        bool
	TimeInForce         uint8
	SelfTradePrevention uint8
	DelegatedPublicKey  string
	ClientOrderId       string
}

const (
	unusedCallbackRate                 = "0.00000000"
	unusedConditionalOrderId           = 0
	unusedIsLiquidationAcquisitionOnly = false
)

// Field order is part of the EIP-712 hash: byte-identical to the "Position" column for type
// `Order` in API_NOTES.md Section B, all 17 fields including the three the docs mark unused —
// those still occupy their slot in the hashed message, they are just not caller-configurable.
var orderType = []apitypes.Type{
	{Name: "nonce", Type: "uint128"},
	{Name: "wallet", Type: "address"},
	{Name: "marketSymbol", Type: "string"},
	{Name: "orderType", Type: "uint8"},
	{Name: "orderSide", Type: "uint8"},
	{Name: "quantity", Type: "string"},
	{Name: "limitPrice", Type: "string"},
	{Name: "triggerPrice", Type: "string"},
	{Name: "triggerType", Type: "uint8"},
	{Name: "callbackRate", Type: "string"},
	{Name: "conditionalOrderId", Type: "uint128"},
	{Name: "isReduceOnly", Type: "bool"},
	{Name: "timeInForce", Type: "uint8"},
	{Name: "selfTradePrevention", Type: "uint8"},
	{Name: "isLiquidationAcquisitionOnly", Type: "bool"},
	{Name: "delegatedPublicKey", Type: "address"},
	{Name: "clientOrderId", Type: "string"},
}

func (s *katanaSigner) typedData(primary string, types []apitypes.Type, message apitypes.TypedDataMessage) apitypes.TypedData {
	return apitypes.TypedData{
		Types:       apitypes.Types{"EIP712Domain": domainType, primary: types},
		PrimaryType: primary,
		Domain:      s.domain(),
		Message:     message,
	}
}

func (s *katanaSigner) hashTypedData(td apitypes.TypedData) ([]byte, error) {
	hash, _, err := apitypes.TypedDataAndHash(td)
	return hash, err
}

func (s *katanaSigner) signTypedData(td apitypes.TypedData) (string, error) {
	hash, err := s.hashTypedData(td)
	if err != nil {
		return "", err
	}
	key, err := crypto.HexToECDSA(trim0x(s.privKeyHex))
	if err != nil {
		return "", err
	}
	sig, err := crypto.Sign(hash, key)
	if err != nil {
		return "", err
	}
	sig[64] += 27 // Ethereum v convention
	return hexutil.Encode(sig), nil
}

func trim0x(s string) string {
	if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
		return s[2:]
	}
	return s
}

// delegatedAddress derives the address of the delegated key that signs every EIP-712 payload, and
// is also the single source of the REST `delegatedKey` parameter, so the signed value and the wire
// value cannot drift apart. privKeyHex is always the delegated key's private key: Katana
// engineering confirmed delegated keys ARE honored by the public API, making the docs' "Unused by
// the public API, always 0x0" note on this field a documentation bug (API_NOTES.md, "Amendments
// from Katana engineering (2026-08-03)", item 1).
func (s *katanaSigner) delegatedAddress() (string, error) {
	key, err := crypto.HexToECDSA(trim0x(s.privKeyHex))
	if err != nil {
		return "", err
	}
	return crypto.PubkeyToAddress(key.PublicKey).Hex(), nil
}

func (s *katanaSigner) orderTypedData(o orderSignPayload) (apitypes.TypedData, error) {
	delegated, err := s.delegatedAddress()
	if err != nil {
		return apitypes.TypedData{}, err
	}
	if o.DelegatedPublicKey != "" && !strings.EqualFold(o.DelegatedPublicKey, delegated) {
		return apitypes.TypedData{}, fmt.Errorf(
			"katana: orderSignPayload.DelegatedPublicKey %s does not match the client's delegated key address %s; leave it empty to derive it automatically",
			o.DelegatedPublicKey, delegated,
		)
	}
	return s.typedData("Order", orderType, apitypes.TypedDataMessage{
		"nonce":                        (*math.HexOrDecimal256)(o.Nonce),
		"wallet":                       o.Wallet,
		"marketSymbol":                 o.MarketSymbol,
		"orderType":                    big.NewInt(int64(o.OrderType)),
		"orderSide":                    big.NewInt(int64(o.OrderSide)),
		"quantity":                     o.Quantity,
		"limitPrice":                   o.LimitPrice,
		"triggerPrice":                 o.TriggerPrice,
		"triggerType":                  big.NewInt(int64(o.TriggerType)),
		"callbackRate":                 unusedCallbackRate,
		"conditionalOrderId":           big.NewInt(unusedConditionalOrderId),
		"isReduceOnly":                 o.IsReduceOnly,
		"timeInForce":                  big.NewInt(int64(o.TimeInForce)),
		"selfTradePrevention":          big.NewInt(int64(o.SelfTradePrevention)),
		"isLiquidationAcquisitionOnly": unusedIsLiquidationAcquisitionOnly,
		"delegatedPublicKey":           delegated,
		"clientOrderId":                o.ClientOrderId,
	}), nil
}

func (s *katanaSigner) orderHash(o orderSignPayload) ([]byte, error) {
	td, err := s.orderTypedData(o)
	if err != nil {
		return nil, err
	}
	return s.hashTypedData(td)
}

func (s *katanaSigner) signOrder(o orderSignPayload) (string, error) {
	td, err := s.orderTypedData(o)
	if err != nil {
		return "", err
	}
	return s.signTypedData(td)
}

// Field order matches API_NOTES.md Section B, type `OrderCancellationByOrderId`.
var cancelByOrderIdsType = []apitypes.Type{
	{Name: "nonce", Type: "uint128"},
	{Name: "wallet", Type: "address"},
	{Name: "delegatedKey", Type: "address"},
	{Name: "orderIds", Type: "string[]"},
}

func (s *katanaSigner) cancelByOrderIdsTypedData(nonce *big.Int, wallet string, orderIDs []string) (apitypes.TypedData, error) {
	delegated, err := s.delegatedAddress()
	if err != nil {
		return apitypes.TypedData{}, err
	}
	return s.typedData("OrderCancellationByOrderId", cancelByOrderIdsType, apitypes.TypedDataMessage{
		"nonce":        (*math.HexOrDecimal256)(nonce),
		"wallet":       wallet,
		"delegatedKey": delegated,
		"orderIds":     orderIDs,
	}), nil
}

func (s *katanaSigner) cancelByOrderIdsHash(nonce *big.Int, wallet string, orderIDs []string) ([]byte, error) {
	td, err := s.cancelByOrderIdsTypedData(nonce, wallet, orderIDs)
	if err != nil {
		return nil, err
	}
	return s.hashTypedData(td)
}

func (s *katanaSigner) signCancelByOrderIds(nonce *big.Int, wallet string, orderIDs []string) (string, error) {
	td, err := s.cancelByOrderIdsTypedData(nonce, wallet, orderIDs)
	if err != nil {
		return "", err
	}
	return s.signTypedData(td)
}

// Field order matches API_NOTES.md Section B, type `OrderCancellationByWallet`. The docs' fourth
// cancellation type, `OrderCancellationByDelegatedKey`, is deliberately not implemented.
var cancelByWalletType = []apitypes.Type{
	{Name: "nonce", Type: "uint128"},
	{Name: "wallet", Type: "address"},
	{Name: "delegatedKey", Type: "address"},
}

func (s *katanaSigner) cancelByWalletTypedData(nonce *big.Int, wallet string) (apitypes.TypedData, error) {
	delegated, err := s.delegatedAddress()
	if err != nil {
		return apitypes.TypedData{}, err
	}
	return s.typedData("OrderCancellationByWallet", cancelByWalletType, apitypes.TypedDataMessage{
		"nonce":        (*math.HexOrDecimal256)(nonce),
		"wallet":       wallet,
		"delegatedKey": delegated,
	}), nil
}

func (s *katanaSigner) cancelByWalletHash(nonce *big.Int, wallet string) ([]byte, error) {
	td, err := s.cancelByWalletTypedData(nonce, wallet)
	if err != nil {
		return nil, err
	}
	return s.hashTypedData(td)
}

// signCancelByWallet and signCancelByMarket sign the two cancel-all variants. No action calls
// either yet — the connector only ever cancels by order id — but both are kept signed and
// golden-vector tested so a future cancel-all does not have to re-derive an EIP-712 type block
// under time pressure.
func (s *katanaSigner) signCancelByWallet(nonce *big.Int, wallet string) (string, error) {
	td, err := s.cancelByWalletTypedData(nonce, wallet)
	if err != nil {
		return "", err
	}
	return s.signTypedData(td)
}

// Field order matches API_NOTES.md Section B, type `OrderCancellationByMarketSymbol`.
var cancelByMarketType = []apitypes.Type{
	{Name: "nonce", Type: "uint128"},
	{Name: "wallet", Type: "address"},
	{Name: "delegatedKey", Type: "address"},
	{Name: "marketSymbol", Type: "string"},
}

func (s *katanaSigner) cancelByMarketTypedData(nonce *big.Int, wallet, market string) (apitypes.TypedData, error) {
	delegated, err := s.delegatedAddress()
	if err != nil {
		return apitypes.TypedData{}, err
	}
	return s.typedData("OrderCancellationByMarketSymbol", cancelByMarketType, apitypes.TypedDataMessage{
		"nonce":        (*math.HexOrDecimal256)(nonce),
		"wallet":       wallet,
		"delegatedKey": delegated,
		"marketSymbol": market,
	}), nil
}

func (s *katanaSigner) cancelByMarketHash(nonce *big.Int, wallet, market string) ([]byte, error) {
	td, err := s.cancelByMarketTypedData(nonce, wallet, market)
	if err != nil {
		return nil, err
	}
	return s.hashTypedData(td)
}

func (s *katanaSigner) signCancelByMarket(nonce *big.Int, wallet, market string) (string, error) {
	td, err := s.cancelByMarketTypedData(nonce, wallet, market)
	if err != nil {
		return "", err
	}
	return s.signTypedData(td)
}

// Field order matches API_NOTES.md Section B, type `InitialMarginFractionOverrideSettings`. The
// field is `initialMarginFractionOverride`, not `initialMarginFraction`.
var imfOverrideType = []apitypes.Type{
	{Name: "nonce", Type: "uint128"},
	{Name: "wallet", Type: "address"},
	{Name: "marketSymbol", Type: "string"},
	{Name: "initialMarginFractionOverride", Type: "string"},
	{Name: "delegatedPublicKey", Type: "address"},
}

func (s *katanaSigner) imfOverrideTypedData(nonce *big.Int, wallet, market, imf string) (apitypes.TypedData, error) {
	delegated, err := s.delegatedAddress()
	if err != nil {
		return apitypes.TypedData{}, err
	}
	return s.typedData("InitialMarginFractionOverrideSettings", imfOverrideType, apitypes.TypedDataMessage{
		"nonce":                         (*math.HexOrDecimal256)(nonce),
		"wallet":                        wallet,
		"marketSymbol":                  market,
		"initialMarginFractionOverride": imf,
		"delegatedPublicKey":            delegated,
	}), nil
}

func (s *katanaSigner) imfOverrideHash(nonce *big.Int, wallet, market, imf string) ([]byte, error) {
	td, err := s.imfOverrideTypedData(nonce, wallet, market, imf)
	if err != nil {
		return nil, err
	}
	return s.hashTypedData(td)
}

func (s *katanaSigner) signIMFOverride(nonce *big.Int, wallet, market, imf string) (string, error) {
	td, err := s.imfOverrideTypedData(nonce, wallet, market, imf)
	if err != nil {
		return "", err
	}
	return s.signTypedData(td)
}
