package katana

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sumex-io/sumex-tradelib/utils"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// referenceHMAC is the assertion side of every signing test below: HMAC-SHA256, hex, computed here
// from crypto/hmac rather than from anything in the package under test.
func referenceHMAC(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// TestCreateSignMatchesReferenceHMAC is the port of the source's TestSignPayloadMatchesReferenceHMAC.
// The standalone signPayload helper is gone — the HMAC is now produced by the createSign request
// option, which is therefore what is asserted.
func TestCreateSignMatchesReferenceHMAC(t *testing.T) {
	payload := `{"nonce":"a","wallet":"0x1"}`
	r := &utils.Request{
		Method:     http.MethodPost,
		Endpoint:   "/v1/orders",
		SecType:    utils.SecTypeSigned,
		BodyString: payload,
		TmpSig:     "topsecret",
	}

	if err := createSign(r); err != nil {
		t.Fatal(err)
	}

	if want := referenceHMAC("topsecret", payload); r.Sign != want {
		t.Fatalf("createSign = %s, want %s", r.Sign, want)
	}
	if r.TmpSig != "" {
		t.Fatal("createSign must clear the secret off the request once it has been consumed")
	}
}

func TestNewNonceProducesUUIDv1AndMatchingUint128(t *testing.T) {
	s, n, err := newNonce()
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 36 {
		t.Fatalf("nonce string = %q, want canonical 36-char UUID", s)
	}
	if s[14] != '1' {
		t.Fatalf("nonce version nibble = %q, want '1' (UUID v1)", s[14])
	}
	if n.Cmp(big.NewInt(0)) <= 0 {
		t.Fatalf("nonce uint128 = %s, want > 0", n)
	}
	if n.BitLen() > 128 {
		t.Fatalf("nonce uint128 has %d bits, want <= 128", n.BitLen())
	}
}

func TestPostSignsExactRequestBodyBytes(t *testing.T) {
	const apiKey = "key"
	const apiSecret = "topsecret"

	var gotAPIKey, gotSignature string
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("KP-API-KEY")
		gotSignature = r.Header.Get("KP-HMAC-SIGNATURE")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewFuturesClient(apiKey, apiSecret, "")
	c.BaseURL = server.URL

	r := &utils.Request{
		Method:     http.MethodPost,
		Endpoint:   "/orders",
		SecType:    utils.SecTypeSigned,
		BodyString: `{"nonce":"a","wallet":"0x1"}`,
	}

	if _, _, err := c.callAPI(context.Background(), r); err != nil {
		t.Fatal(err)
	}

	if gotAPIKey != apiKey {
		t.Fatalf("KP-API-KEY = %q, want %q", gotAPIKey, apiKey)
	}
	if want := referenceHMAC(apiSecret, string(gotBody)); gotSignature != want {
		t.Fatalf("KP-HMAC-SIGNATURE = %s, want %s (HMAC over received body %s)", gotSignature, want, gotBody)
	}
}

// TestGetOmitsTheQuerySeparatorWhenThereIsNoQuery: a GET carrying no query at all must not append
// a bare "?" to the request URL. The HMAC is signed over the query string either way, so the
// signature must be unchanged — that is asserted here too, not assumed. The request is built
// inline because the subject is the signed transport path, not any particular endpoint.
func TestGetOmitsTheQuerySeparatorWhenThereIsNoQuery(t *testing.T) {
	var gotRawQuery, gotRequestURI, gotSignature string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		gotRequestURI = r.RequestURI
		gotSignature = r.Header.Get("KP-HMAC-SIGNATURE")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewFuturesClient("key", "topsecret", "")
	c.BaseURL = server.URL

	signedNoQuery := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v1/markets",
		SecType:  utils.SecTypeSigned,
	}
	if _, _, err := c.callAPI(context.Background(), signedNoQuery); err != nil {
		t.Fatal(err)
	}
	if gotRequestURI != "/v1/markets" {
		t.Fatalf("request URI = %q, want /v1/markets with no trailing \"?\"", gotRequestURI)
	}
	if gotRawQuery != "" {
		t.Fatalf("raw query = %q, want empty", gotRawQuery)
	}
	if want := referenceHMAC("topsecret", ""); gotSignature != want {
		t.Fatalf("KP-HMAC-SIGNATURE = %s, want %s (HMAC over the empty query string, unchanged by dropping the \"?\")", gotSignature, want)
	}
}

// TestNon2xxErrorCarriesTheStatusAndABoundedBody: the error string reaches both the API caller and
// any on-disk log the caller keeps, so a 5xx HTML page or proxy dump must not land there unbounded.
func TestNon2xxErrorCarriesTheStatusAndABoundedBody(t *testing.T) {
	huge := strings.Repeat("A", 5000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>\n<body>\n" + huge + "\n</body>\n</html>"))
	}))
	defer server.Close()

	c := NewFuturesClient("key", "topsecret", "")
	c.BaseURL = server.URL

	_, _, err := c.callAPI(context.Background(), marketsRequest())
	if err == nil {
		t.Fatal("expected a non-2xx response to be an error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "HTTP 502") {
		t.Fatalf("error = %q, want it to name the HTTP status", msg)
	}
	// 512 body bytes + the "... (truncated)" marker + the "katana: HTTP 502: " prefix is
	// comfortably under 600; the raw body alone was over 5000.
	if len(msg) > 600 {
		t.Fatalf("error message is %d bytes, want it bounded well under the 5000-byte upstream body", len(msg))
	}
	if !strings.Contains(msg, "(truncated)") {
		t.Fatalf("error = %q, want a truncated body to say so", msg)
	}
	if strings.Contains(msg, "\n") {
		t.Fatalf("error = %q, want a single-line message", msg)
	}
}

// TestNon2xxErrorKeepsAShortBodyIntact makes sure the bound does not mangle the small
// {"code","message"} JSON Katana actually documents for its own errors.
func TestNon2xxErrorKeepsAShortBodyIntact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":"INVALID_API_KEY","message":"Invalid API key"}`))
	}))
	defer server.Close()

	c := NewFuturesClient("key", "topsecret", "")
	c.BaseURL = server.URL

	_, _, err := c.callAPI(context.Background(), marketsRequest())
	if err == nil {
		t.Fatal("expected a non-2xx response to be an error")
	}
	want := `katana: HTTP 401: {"code":"INVALID_API_KEY","message":"Invalid API key"}`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

// TestRateLimitIsReportedAsSuch pins the one status with its own message.
func TestRateLimitIsReportedAsSuch(t *testing.T) {
	err := aPIError{StatusCode: http.StatusTooManyRequests, Raw: []byte(`{"code":"RATE_LIMIT"}`)}
	if err.Error() != "katana rate limit exceeded" {
		t.Fatalf("error = %q, want the dedicated rate-limit message", err.Error())
	}
}

func TestGetSignsExactQueryStringBytes(t *testing.T) {
	const apiKey = "key"
	const apiSecret = "topsecret"

	var gotAPIKey, gotSignature, gotRawQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("KP-API-KEY")
		gotSignature = r.Header.Get("KP-HMAC-SIGNATURE")
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewFuturesClient(apiKey, apiSecret, "")
	c.BaseURL = server.URL

	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/positions",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(utils.Params{"symbol": "BTC-USD", "wallet": "0x1"})

	if _, _, err := c.callAPI(context.Background(), r); err != nil {
		t.Fatal(err)
	}

	if gotAPIKey != apiKey {
		t.Fatalf("KP-API-KEY = %q, want %q", gotAPIKey, apiKey)
	}
	if want := referenceHMAC(apiSecret, gotRawQuery); gotSignature != want {
		t.Fatalf("KP-HMAC-SIGNATURE = %s, want %s (HMAC over received query %q)", gotSignature, want, gotRawQuery)
	}
}

// ===============EIP-712=================

// Deterministic throwaway key; never used outside tests.
const testPrivKey = "4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"

func testSigner(demo bool) *katanaSigner {
	c := NewFuturesClient("k", "s", testPrivKey)
	c.SetDemo(demo)
	return c.signer()
}

func TestSignOrderIsDeterministicAndRecoverable(t *testing.T) {
	s := testSigner(false)
	payload := orderSignPayload{
		Nonce:               big.NewInt(1234567890),
		Wallet:              "0x1111111111111111111111111111111111111111",
		MarketSymbol:        "ETH-USD",
		OrderType:           1,
		OrderSide:           0,
		Quantity:            "1.00000000",
		LimitPrice:          "2300.00000000",
		TriggerPrice:        "0.00000000",
		TriggerType:         0,
		IsReduceOnly:        false,
		TimeInForce:         0,
		SelfTradePrevention: 0,
		DelegatedPublicKey:  "", // empty = derive from the client's key automatically
		ClientOrderId:       "abc",
	}

	first, err := s.signOrder(payload)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.signOrder(payload)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("signature not deterministic: %s vs %s", first, second)
	}
	if !strings.HasPrefix(first, "0x") || len(first) != 132 {
		t.Fatalf("signature = %s, want 0x-prefixed 65-byte hex", first)
	}

	hash, err := s.orderHash(payload)
	if err != nil {
		t.Fatal(err)
	}
	assertRecoversToDelegatedKey(t, hash, first)
}

// TestDemoFlagMovesTheHostAndTheDomainTogether is the port of the source's TestDomainSwitchesOnSandbox
// plus TestBaseURLSwitchesOnDemo: they were two tests over one flag, and in this layout the flag is
// literally the same setter, so they are asserted together. Signing for one environment and sending
// to the other must stay inexpressible.
func TestDemoFlagMovesTheHostAndTheDomainTogether(t *testing.T) {
	prodClient := NewFuturesClient("k", "s", testPrivKey)
	if prodClient.BaseURL != "https://api-perps.katana.network" {
		t.Fatalf("prod base = %s", prodClient.BaseURL)
	}
	prod := prodClient.signer().domain()

	sandboxClient := NewFuturesClient("k", "s", testPrivKey)
	sandboxClient.SetDemo(true)
	if sandboxClient.BaseURL != "https://api-perps-sandbox.katana.network" {
		t.Fatalf("sandbox base = %s", sandboxClient.BaseURL)
	}
	sandbox := sandboxClient.signer().domain()

	if prod.Name != "KatanaPerps" {
		t.Fatalf("prod domain Name = %q, want %q", prod.Name, "KatanaPerps")
	}
	if prod.Version != "2.0.0" {
		t.Fatalf("prod domain Version = %q, want %q", prod.Version, "2.0.0")
	}
	if (*big.Int)(prod.ChainId).Int64() != 747474 {
		t.Fatalf("prod domain ChainId = %s, want 747474", (*big.Int)(prod.ChainId))
	}
	if prod.VerifyingContract != "0x62230CeA619F734cc215bB8074bbF07bE4Eb633e" {
		t.Fatalf("prod domain VerifyingContract = %q, want %q", prod.VerifyingContract, "0x62230CeA619F734cc215bB8074bbF07bE4Eb633e")
	}

	if sandbox.Name != "KatanaPerps" {
		t.Fatalf("sandbox domain Name = %q, want %q", sandbox.Name, "KatanaPerps")
	}
	if sandbox.Version != "2.0.0-sandbox" {
		t.Fatalf("sandbox domain Version = %q, want %q", sandbox.Version, "2.0.0-sandbox")
	}
	if (*big.Int)(sandbox.ChainId).Int64() != 737373 {
		t.Fatalf("sandbox domain ChainId = %s, want 737373", (*big.Int)(sandbox.ChainId))
	}
	if sandbox.VerifyingContract != "0x92d3072dDe1aD3e9B7895500F504aA5e664E71d3" {
		t.Fatalf("sandbox domain VerifyingContract = %q, want %q", sandbox.VerifyingContract, "0x92d3072dDe1aD3e9B7895500F504aA5e664E71d3")
	}
}

// ---------------------------------------------------------------------------
// Golden vectors.
//
// These lock the exact wire format this connector exists to get right: the type declarations
// (orderType / cancelByOrderIdsType / imfOverrideType), the field order within each, the Solidity
// type string of every field, and the domain separator — for the fixed test private key below,
// against a fixed, fully-specified payload for each of the signed struct shapes this package
// implements.
//
// Each hash/signature pair below was computed once against the original connector and frozen
// verbatim; they are unchanged by this port, which is the point — a foundation that reshapes the
// files must not reshape the bytes. The recoverability re-check (SigToPub back to the delegated
// key's address) re-derives from first principles that the frozen signature really was produced
// over the frozen hash, so a single copy-paste error in a literal cannot go unnoticed.
//
// A determinism test (TestSignOrderIsDeterministicAndRecoverable above) cannot catch drift in the
// type block or field order: it recomputes the "expected" hash from the same orderTypedData it is
// testing, so reordering fields, widening a uint128 to uint256, or dropping one of the "unused,
// always <constant>" Order fields moves both sides of that assertion together and the test stays
// green. These golden vectors compare against values frozen independently of the current code, so
// they catch exactly that class of silent drift.
//
// If any of these fail after a code change, the wire format changed. Do NOT "fix" the test by
// re-freezing the new value — re-validate the new value against Katana's sandbox first.
func TestSignOrderGoldenVector(t *testing.T) {
	s := testSigner(false)
	delegated, err := s.delegatedAddress()
	if err != nil {
		t.Fatal(err)
	}

	payload := orderSignPayload{
		Nonce:               big.NewInt(1234567890),
		Wallet:              "0x1111111111111111111111111111111111111111",
		MarketSymbol:        "ETH-USD",
		OrderType:           1,
		OrderSide:           0,
		Quantity:            "1.00000000",
		LimitPrice:          "2300.00000000",
		TriggerPrice:        "0.00000000",
		TriggerType:         0,
		IsReduceOnly:        false,
		TimeInForce:         0,
		SelfTradePrevention: 0,
		DelegatedPublicKey:  delegated, // matches the client's key: exercises the "matches" branch
		ClientOrderId:       "abc",
	}

	const wantHash = "0x1548fab064d23dfbfeb81fbe8a1b555990977ba9f08e869e1f753a20bc55f759"
	const wantSig = "0x1bd240a91fdf514022ed3692973d7bf5e77fc5a73876d9a0f91dcc8c16ca7c3c04fc34295019dbd6ed28096b88eb70e0afb7c71114a81c1cc9c5d038d80e165c1b"

	assertGoldenVector(t, wantHash, wantSig, func() ([]byte, error) { return s.orderHash(payload) }, func() (string, error) { return s.signOrder(payload) })
}

func TestSignOrderDelegatedPublicKeyMismatchErrors(t *testing.T) {
	s := testSigner(false)
	payload := orderSignPayload{
		Nonce:               big.NewInt(1234567890),
		Wallet:              "0x1111111111111111111111111111111111111111",
		MarketSymbol:        "ETH-USD",
		OrderType:           1,
		OrderSide:           0,
		Quantity:            "1.00000000",
		LimitPrice:          "2300.00000000",
		TriggerPrice:        "0.00000000",
		TriggerType:         0,
		IsReduceOnly:        false,
		TimeInForce:         0,
		SelfTradePrevention: 0,
		DelegatedPublicKey:  "0x9999999999999999999999999999999999999999", // not the client's delegated key
		ClientOrderId:       "abc",
	}

	if _, err := s.signOrder(payload); err == nil {
		t.Fatal("signOrder = nil error, want an error for a DelegatedPublicKey that does not match the client's derived address")
	}
}

func TestSignOrderDelegatedPublicKeyCaseInsensitiveMatch(t *testing.T) {
	s := testSigner(false)
	delegated, err := s.delegatedAddress()
	if err != nil {
		t.Fatal(err)
	}

	payload := orderSignPayload{
		Nonce:               big.NewInt(1),
		Wallet:              "0x1111111111111111111111111111111111111111",
		MarketSymbol:        "ETH-USD",
		OrderType:           0,
		OrderSide:           0,
		Quantity:            "1.00000000",
		LimitPrice:          "0.00000000",
		TriggerPrice:        "0.00000000",
		TriggerType:         0,
		IsReduceOnly:        false,
		TimeInForce:         0,
		SelfTradePrevention: 0,
		DelegatedPublicKey:  strings.ToLower(delegated), // same address, different casing
		ClientOrderId:       "",
	}

	if _, err := s.signOrder(payload); err != nil {
		t.Fatalf("signOrder with a case-different but matching DelegatedPublicKey = %v, want success", err)
	}
}

func TestSignCancelByOrderIdsGoldenVectorAndRecovers(t *testing.T) {
	s := testSigner(false)
	nonce := big.NewInt(987654321)
	wallet := "0x2222222222222222222222222222222222222222"
	orderIDs := []string{"3a9ef9c0-a779-11ea-907d-23e999279287", "client:199283"}

	const wantHash = "0x87395e6ac2fb6ff9627ed5743fa2e2fa02bc79cfa90e857e6c6aa7339864dccd"
	const wantSig = "0xe4733ea5611a6e9bd567db2f3e679b758f3a1b112ebbaccc2ef7a1f7f3e07b3f73b7d64d3a7b1e2cfd62fe1d8c8f2196a00ac05d24f886427b114ab9a24717ba1c"

	assertGoldenVector(t, wantHash, wantSig,
		func() ([]byte, error) { return s.cancelByOrderIdsHash(nonce, wallet, orderIDs) },
		func() (string, error) { return s.signCancelByOrderIds(nonce, wallet, orderIDs) },
	)
}

func TestSignCancelByMarketRecovers(t *testing.T) {
	s := testSigner(false)
	nonce := big.NewInt(555)
	wallet := "0x3333333333333333333333333333333333333333"
	market := "ETH-USD"

	sig, err := s.signCancelByMarket(nonce, wallet, market)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sig, "0x") || len(sig) != 132 {
		t.Fatalf("signature = %s, want 0x-prefixed 65-byte hex", sig)
	}

	hash, err := s.cancelByMarketHash(nonce, wallet, market)
	if err != nil {
		t.Fatal(err)
	}
	assertRecoversToDelegatedKey(t, hash, sig)
}

// TestSignCancelByWalletGoldenVectorAndRecovers: every other signer has a golden vector (see the
// "Golden vectors" doc comment above TestSignOrderGoldenVector for why a pure determinism/recovery
// test cannot catch field-order or type-string drift on its own).
func TestSignCancelByWalletGoldenVectorAndRecovers(t *testing.T) {
	s := testSigner(false)
	nonce := big.NewInt(13579)
	wallet := "0x6666666666666666666666666666666666666666"

	const wantHash = "0x6fc2b1c4558cdc392c7ae68cdc6d159925bbd255ec684b7051c07fe6d753ca8c"
	const wantSig = "0x9a59150422d155070749e11578c8a4c8db5f8da298fcb69c0850a6fae2c7e94a3b4a85856ed230657e503ec9205c616e110073f01ad65fbdc8114a4fcfc63d7b1b"

	assertGoldenVector(t, wantHash, wantSig,
		func() ([]byte, error) { return s.cancelByWalletHash(nonce, wallet) },
		func() (string, error) { return s.signCancelByWallet(nonce, wallet) },
	)
}

func TestSignCancelByWalletDiffersFromCancelByMarketForSameNonceAndWallet(t *testing.T) {
	s := testSigner(false)
	nonce := big.NewInt(1)
	wallet := "0x5555555555555555555555555555555555555555"

	byWallet, err := s.signCancelByWallet(nonce, wallet)
	if err != nil {
		t.Fatal(err)
	}
	byMarket, err := s.signCancelByMarket(nonce, wallet, "ETH-USD")
	if err != nil {
		t.Fatal(err)
	}
	if byWallet == byMarket {
		t.Fatal("signCancelByWallet and signCancelByMarket produced the same signature for the same nonce/wallet — the EIP-712 type identifier is not distinguishing the two struct shapes")
	}
}

func TestSignIMFOverrideGoldenVectorAndRecovers(t *testing.T) {
	s := testSigner(false)
	nonce := big.NewInt(987654321)
	wallet := "0x2222222222222222222222222222222222222222"
	market := "BTC-USD"
	imf := "0.50000000"

	const wantHash = "0x2ccf278a44d478cb734d56bb99964d04b98344e18bf35d59898fd05c2b20a143"
	const wantSig = "0x4d1f8608da8128f554d530e3f72cdd6d96228794568f12f20deacffc7be3fdcd4ebd088f063e9b38b5a1f0afd7b20e3ce1f49a3d65aa1936859de950b8a1dacb1c"

	assertGoldenVector(t, wantHash, wantSig,
		func() ([]byte, error) { return s.imfOverrideHash(nonce, wallet, market, imf) },
		func() (string, error) { return s.signIMFOverride(nonce, wallet, market, imf) },
	)
}

// assertGoldenVector asserts that hashFn/signFn reproduce the frozen hash and signature exactly,
// then independently re-derives the delegated key's address from the frozen hash+signature via
// ECDSA recovery, so a matching literal pair can't have been frozen from two different runs.
func assertGoldenVector(t *testing.T, wantHash, wantSig string, hashFn func() ([]byte, error), signFn func() (string, error)) {
	t.Helper()

	gotHash, err := hashFn()
	if err != nil {
		t.Fatal(err)
	}
	if got := hexutil.Encode(gotHash); got != wantHash {
		t.Fatalf("hash = %s, want frozen golden value %s (wire format changed: re-validate against sandbox before re-freezing)", got, wantHash)
	}

	gotSig, err := signFn()
	if err != nil {
		t.Fatal(err)
	}
	if gotSig != wantSig {
		t.Fatalf("signature = %s, want frozen golden value %s (wire format changed: re-validate against sandbox before re-freezing)", gotSig, wantSig)
	}

	assertRecoversToDelegatedKey(t, gotHash, gotSig)
}

func assertRecoversToDelegatedKey(t *testing.T, hash []byte, sig string) {
	t.Helper()

	sigBytes := common0xToBytes(t, sig)
	sigBytes[64] -= 27
	pub, err := crypto.SigToPub(hash, sigBytes)
	if err != nil {
		t.Fatal(err)
	}
	key, err := crypto.HexToECDSA(testPrivKey)
	if err != nil {
		t.Fatal(err)
	}
	if crypto.PubkeyToAddress(*pub) != crypto.PubkeyToAddress(key.PublicKey) {
		t.Fatal("recovered signer does not match the delegated key")
	}
}

func common0xToBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hexutil.Decode(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
