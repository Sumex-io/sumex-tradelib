package hyperliquid

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/Sumex-io/sumex-tradelib/utils"
)

func loadFixture(t *testing.T, name string, out interface{}) {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatal(err)
	}
}

func TestConvertUnifiedBalanceUsesSpotUSDCWhenPerpsWithdrawableIsEmpty(t *testing.T) {
	perps := futures_clearinghouseState{}
	spot := spot_spotClearinghouseState{}
	loadFixture(t, "unified_clearinghouse_state.json", &perps)
	loadFixture(t, "unified_spot_clearinghouse_state.json", &spot)

	got := (&futures_converts{}).convertUnifiedBalance(perps, spot)

	if len(got) != 1 || got[0].Asset != "USDC" {
		t.Fatalf("got %+v, want one USDC row", got)
	}
	for field, pair := range map[string][2]float64{
		"available": {utils.StringToFloat(got[0].Available), 385.72516664 - 33.534323},
		"balance":   {utils.StringToFloat(got[0].Balance), 385.72516664 + 0.4466},
	} {
		if math.Abs(pair[0]-pair[1]) > 1e-9 {
			t.Fatalf("%s = %v, want %v", field, pair[0], pair[1])
		}
	}
}

func TestConvertUnifiedBalanceNeverGoesNegative(t *testing.T) {
	spot := spot_spotClearinghouseState{}
	if err := json.Unmarshal([]byte(`{"balances":[{"coin":"USDC","token":0,"total":"10","hold":"12"}]}`), &spot); err != nil {
		t.Fatal(err)
	}

	got := (&futures_converts{}).convertUnifiedBalance(futures_clearinghouseState{}, spot)

	if got[0].Available != "0" {
		t.Fatalf("available = %s, want 0", got[0].Available)
	}
}
