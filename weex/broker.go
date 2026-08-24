package weex

import (
	"strings"

	"github.com/google/uuid"
)

// maxClientOrderIDBytes is WEEX's cap on newClientOrderId; the broker prefix spends the same budget.
const maxClientOrderIDBytes = 64

// resolveClientOrderID stamps the broker code onto the client order id: WEEX credits broker volume
// only for ids that read `b-{brokerId}-...`. An unset broker leaves the id alone, and an id that
// already carries the prefix is not stamped twice.
func resolveClientOrderID(brokerID, base string) string {
	broker := strings.TrimSpace(brokerID)
	if broker == "" {
		return base
	}

	prefix := "b-" + broker + "-"
	if strings.HasPrefix(base, prefix) {
		return truncateClientOrderID(base)
	}

	suffix := strings.TrimSpace(base)
	if suffix == "" {
		suffix = uuid.NewString()
	}

	return truncateClientOrderID(prefix + suffix)
}

// truncateClientOrderID keeps an id inside the cap so the prefix cannot push it over and get the
// whole order rejected.
func truncateClientOrderID(id string) string {
	if len(id) <= maxClientOrderIDBytes {
		return id
	}
	return id[:maxClientOrderIDBytes]
}
