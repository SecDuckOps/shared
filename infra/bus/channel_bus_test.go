package bus_test

import (
	"testing"

	"github.com/SecDuckOps/shared/infra/bus"
	bustest "github.com/SecDuckOps/shared/infra/bus/testing"
)

func TestChannelBus_Contract(t *testing.T) {
	b := bus.NewChannelBus()
	bustest.RunEventBusContract(t, b)
}
