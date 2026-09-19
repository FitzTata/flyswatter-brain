package neural

import (
	"testing"

	"github.com/FitzTata/flyswatter-brain/internal/game"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    Mode
		wantErr bool
	}{
		{name: "default", raw: "", want: ModeShared},
		{name: "shared", raw: "shared", want: ModeShared},
		{name: "static rejected", raw: "static", wantErr: true},
		{name: "random rejected", raw: "random", wantErr: true},
		{name: "invalid", raw: "personal", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseMode(tt.raw)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBiasEscape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		action     game.Action
		ticks      int
		wantAction game.Action
		wantTicks  int
	}{
		{name: "no bias", action: game.ActionStraight, ticks: 0, wantAction: game.ActionStraight, wantTicks: 0},
		{name: "straight to escape", action: game.ActionStraight, ticks: 3, wantAction: game.ActionEscape, wantTicks: 2},
		{name: "keep turn", action: game.ActionTurnLeft, ticks: 3, wantAction: game.ActionTurnLeft, wantTicks: 2},
		{name: "keep escape", action: game.ActionEscape, ticks: 1, wantAction: game.ActionEscape, wantTicks: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotAction, gotTicks := biasEscape(tt.action, tt.ticks)

			assert.Equal(t, tt.wantAction, gotAction)
			assert.Equal(t, tt.wantTicks, gotTicks)
		})
	}
}
