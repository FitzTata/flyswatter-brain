package neural

import (
	"testing"

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
		{name: "default", raw: "", want: ModeStatic},
		{name: "static", raw: "static", want: ModeStatic},
		{name: "shared", raw: "shared", want: ModeShared},
		{name: "random", raw: "random", want: ModeRandom},
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
