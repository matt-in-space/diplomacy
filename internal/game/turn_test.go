package game

import "testing"

func TestTurnNext(t *testing.T) {
	tests := []struct {
		name string
		turn Turn
		want Turn
	}{
		{
			name: "accept orders advances to resolve orders",
			turn: Turn{Season: Spring, Phase: AcceptOrders, Year: 1},
			want: Turn{Season: Spring, Phase: ResolveOrders, Year: 1},
		},
		{
			name: "resolve orders advances to accept retreats",
			turn: Turn{Season: Spring, Phase: ResolveOrders, Year: 1},
			want: Turn{Season: Spring, Phase: AcceptRetreats, Year: 1},
		},
		{
			name: "accept retreats advances to resolve retreats",
			turn: Turn{Season: Spring, Phase: AcceptRetreats, Year: 1},
			want: Turn{Season: Spring, Phase: ResolveRetreats, Year: 1},
		},
		{
			name: "spring resolve retreats advances to fall orders",
			turn: Turn{Season: Spring, Phase: ResolveRetreats, Year: 1},
			want: Turn{Season: Fall, Phase: AcceptOrders, Year: 1},
		},
		{
			name: "fall resolve retreats advances to accept adjustments",
			turn: Turn{Season: Fall, Phase: ResolveRetreats, Year: 1},
			want: Turn{Season: Fall, Phase: AcceptAdjustments, Year: 1},
		},
		{
			name: "accept adjustments advances to resolve adjustments",
			turn: Turn{Season: Fall, Phase: AcceptAdjustments, Year: 1},
			want: Turn{Season: Fall, Phase: ResolveAdjustments, Year: 1},
		},
		{
			name: "resolve adjustments advances to next spring",
			turn: Turn{Season: Fall, Phase: ResolveAdjustments, Year: 1},
			want: Turn{Season: Spring, Phase: AcceptOrders, Year: 2},
		},
		{
			name: "completed remains completed",
			turn: Turn{Season: Fall, Phase: Completed, Year: 1},
			want: Turn{Season: Fall, Phase: Completed, Year: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.turn.Next()
			if got != tt.want {
				t.Fatalf("Next() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
