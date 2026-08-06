package katana

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

// Note on the source connector's spot-rejection tests: they asserted that a dispatcher answered
// every real spot action with "Exchange does not support spot trading." This library has no
// dispatcher, and SpotClient exposes only NewGetAccountInfo and NewSignAuthStream — no balance,
// order, position or instrument action exists on it at all. The rejection is therefore structural
// and enforced by the compiler; there is no runtime call left to assert an error string against.
// The two account actions below are exercised on BOTH clients for exactly that reason: they are the
// entire spot surface.

// --- getAccountInfo ---

// TestGetAccountInfoMapsWalletAndHardcodedPermissions asserts against a hand-written literal
// response, not a value recomputed via the code under test.
func TestGetAccountInfoMapsWalletAndHardcodedPermissions(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, singleWalletFixture)
		},
	})
	defer server.Close()

	// testPrivKey (a valid throwaway key) rather than "": getAccountInfo rejects a connection whose
	// delegated key cannot be parsed, and a connection that could never sign an order is not a
	// working connection.
	c := NewSpotClient("k", "s", testPrivKey)
	c.BaseURL = server.URL

	got, err := c.NewGetAccountInfo().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := entity.AccountInformation{
		UID:         "0xWALLET",
		Label:       "",
		IP:          "0.0.0.0/0",
		CanRead:     true,
		CanTrade:    true,
		CanTransfer: false,
		PermSpot:    false,
		PermFutures: true,
	}
	if got.UID != want.UID || got.Label != want.Label || got.IP != want.IP ||
		got.CanRead != want.CanRead || got.CanTrade != want.CanTrade || got.CanTransfer != want.CanTransfer ||
		got.PermSpot != want.PermSpot || got.PermFutures != want.PermFutures {
		t.Fatalf("getAccountInfo() = %+v, want %+v", got, want)
	}
	if len(got.ExtraInfo) != 0 {
		t.Fatalf("ExtraInfo = %+v, want empty (Katana exposes no sub-account identities)", got.ExtraInfo)
	}
}

// TestFuturesGetAccountInfoMatchesTheSpotClient: both clients answer the same account action from
// the same wallet resolution, so a divergence between the twin factories would be a real defect.
func TestFuturesGetAccountInfoMatchesTheSpotClient(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, singleWalletFixture)
		},
	})
	defer server.Close()

	spot := NewSpotClient("k", "s", testPrivKey)
	spot.BaseURL = server.URL
	futures := NewFuturesClient("k", "s", testPrivKey)
	futures.BaseURL = server.URL

	gotSpot, err := spot.NewGetAccountInfo().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	gotFutures, err := futures.NewGetAccountInfo().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotSpot.UID != gotFutures.UID || gotSpot.PermFutures != gotFutures.PermFutures || gotSpot.PermSpot != gotFutures.PermSpot {
		t.Fatalf("spot = %+v, futures = %+v, want identical account information from both clients", gotSpot, gotFutures)
	}
}

// TestGetAccountInfoPropagatesWalletResolutionError ensures a failure resolving the wallet is never
// swallowed into a fabricated "it works" response.
func TestGetAccountInfoPropagatesWalletResolutionError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[]`)
		},
	})
	defer server.Close()

	c := NewSpotClient("k", "s", testPrivKey)
	c.BaseURL = server.URL

	if _, err := c.NewGetAccountInfo().Do(context.Background()); err == nil {
		t.Fatal("expected getAccountInfo to propagate the no-wallet-found error, got nil")
	}
}

// TestGetAccountInfoJSONContract confirms the account response serialises to exactly the key set
// consumers read, using a hand-written literal for the JSON it expects.
func TestGetAccountInfoJSONContract(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, singleWalletFixture)
		},
	})
	defer server.Close()

	c := NewSpotClient("k", "s", testPrivKey)
	c.BaseURL = server.URL

	got, err := c.NewGetAccountInfo().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"uid":"0xWALLET","label":"","ip":"0.0.0.0/0","canRead":true,"canTrade":true,"canTransfer":false,"permSpot":false,"permFutures":true}`
	if string(raw) != want {
		t.Fatalf("got %s, want %s", raw, want)
	}
}

// TestGetAccountInfoRejectsAMalformedDelegatedKey: the delegated key is half of a Katana connection
// (every trade action signs its EIP-712 payload with it). Without this check a user pasting a
// malformed key connects successfully, is badged TRADE, and discovers the problem on their first
// order as a raw crypto.HexToECDSA error. The wallet call still succeeds in this fixture — the
// connection must fail on the key alone, and the message must name it.
func TestGetAccountInfoRejectsAMalformedDelegatedKey(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, singleWalletFixture)
		},
	})
	defer server.Close()

	c := NewSpotClient("k", "s", "not-a-private-key")
	c.BaseURL = server.URL

	_, err := c.NewGetAccountInfo().Do(context.Background())
	if err == nil {
		t.Fatal("expected getAccountInfo to reject a malformed delegated key instead of reporting a working, trade-capable connection")
	}
	if !strings.Contains(err.Error(), "delegated key") {
		t.Fatalf("error = %v, want it to name the delegated key as the problem", err)
	}
}

// TestGetAccountInfoRejectsAnEmptyDelegatedKey covers the other shape of the same misconfiguration:
// no key supplied at all.
func TestGetAccountInfoRejectsAnEmptyDelegatedKey(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, singleWalletFixture)
		},
	})
	defer server.Close()

	c := NewSpotClient("k", "s", "")
	c.BaseURL = server.URL

	if _, err := c.NewGetAccountInfo().Do(context.Background()); err == nil {
		t.Fatal("expected getAccountInfo to reject an empty delegated key")
	}
}

// --- signAuthStream (the WebSocket auth token) ---

const wsTokenFixture = `{"token":"opaque-ws-token-abc123"}`

// TestSignAuthStreamMapsTokenIntoSignatureField asserts against a hand-written literal: the opaque
// Katana "token" field must land in the contract's "signature" field verbatim, not be recomputed or
// reshaped.
func TestSignAuthStreamMapsTokenIntoSignatureField(t *testing.T) {
	var gotWallet, gotNonce string
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, singleWalletFixture)
		},
		"/v1/wsToken": func(w http.ResponseWriter, r *http.Request) {
			gotWallet = r.URL.Query().Get("wallet")
			gotNonce = r.URL.Query().Get("nonce")
			writeJSON(t, w, wsTokenFixture)
		},
	})
	defer server.Close()

	c := NewSpotClient("k", "s", "")
	c.BaseURL = server.URL

	got, err := c.NewSignAuthStream().TimeStamp(1700000000000).Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := entity.SignAuthStream{Signature: "opaque-ws-token-abc123"}
	if got != want {
		t.Fatalf("signAuthStream = %+v, want %+v", got, want)
	}
	if gotWallet != "0xWALLET" {
		t.Fatalf("wallet query param = %q, want 0xWALLET (GET /v1/wsToken must be scoped to the resolved wallet)", gotWallet)
	}
	if gotNonce == "" {
		t.Fatal("nonce query param is empty, want a generated nonce")
	}
}

// TestSignAuthStreamWorksWithoutACallerTimestamp covers the no-timestamp path. Katana mints the
// token server-side with no timestamp input, so the caller's value changes nothing about the
// request or the answer — the token must come back either way.
func TestSignAuthStreamWorksWithoutACallerTimestamp(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, singleWalletFixture)
		},
		"/v1/wsToken": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, wsTokenFixture)
		},
	})
	defer server.Close()

	c := NewSpotClient("k", "s", "")
	c.BaseURL = server.URL

	got, err := c.NewSignAuthStream().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Signature != "opaque-ws-token-abc123" {
		t.Fatalf("Signature = %q, want opaque-ws-token-abc123", got.Signature)
	}
}

// TestSignAuthStreamPropagatesWalletResolutionError ensures a wallet-resolution failure is never
// swallowed.
func TestSignAuthStreamPropagatesWalletResolutionError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[]`)
		},
	})
	defer server.Close()

	c := NewSpotClient("k", "s", "")
	c.BaseURL = server.URL

	if _, err := c.NewSignAuthStream().Do(context.Background()); err == nil {
		t.Fatal("expected signAuthStream to propagate the no-wallet-found error, got nil")
	}
}

// TestSignAuthStreamPropagatesEndpointError ensures a GET /v1/wsToken failure is never swallowed
// into a fabricated token.
func TestSignAuthStreamPropagatesEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, singleWalletFixture)
		},
		"/v1/wsToken": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(t, w, `{"code":"INVALID_API_KEY","message":"Invalid API key"}`)
		},
	})
	defer server.Close()

	c := NewSpotClient("k", "s", "")
	c.BaseURL = server.URL

	if _, err := c.NewSignAuthStream().Do(context.Background()); err == nil {
		t.Fatal("expected signAuthStream to propagate the GET /v1/wsToken error, got nil")
	}
}

// TestFuturesClientSignAuthStreamMatchesTheSpotClient: the futures client exposes the same token
// action, and both must answer identically — the token is account-scoped, not market-scoped.
func TestFuturesClientSignAuthStreamMatchesTheSpotClient(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, singleWalletFixture)
		},
		"/v1/wsToken": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, wsTokenFixture)
		},
	})
	defer server.Close()

	c := NewFuturesClient("k", "s", "")
	c.BaseURL = server.URL

	got, err := c.NewSignAuthStream().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Signature != "opaque-ws-token-abc123" {
		t.Fatalf("Signature = %q, want opaque-ws-token-abc123", got.Signature)
	}
}
