package block

import (
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/transaction/basic"
	"reflect"
	"testing"
)





func init() {
	newCntAccountItem = CntAccountItem{
		"b0001",
		5,
		-1,
	}



}

func Test_nonceHeap_Len(t *testing.T) {
	tests := []struct {
		name string
		h    nonceHp
		want int
	}{
		// TODO: Add test cases.
		{name: "test nonceheapLen",
			h: nonceHp{uint64(10), uint64(13), uint64(9)}, want: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h.Len(); got != tt.want {
				assert.Equal(t, tt.want, got)
				t.Errorf("Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_nonceHeap_Less(t *testing.T) {
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		h    nonceHp
		args args
		want bool
	}{
		// TODO: Add test cases.
		{name: "test nonceheap less", h: nonceHp{uint64(11), uint64(18)},
			args: args{0, 1}, want: true},
		{name: "test nonceheap less", h: nonceHp{uint64(18), uint64(10)},
			args: args{0, 1}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h.Less(tt.args.i, tt.args.j); got != tt.want {
				t.Errorf("Less() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCntAccountHeap_Len(t *testing.T) {
	tests := []struct {
		name string
		pq   CntAccountHeap
		want int
	}{
		// TODO: Add test cases.
		{name: "test for CntAccountHeap_Len",
			pq:   newCntAccountHeap,
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pq.Len(); got != tt.want {
				t.Errorf("Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCntAccountHeap_Less(t *testing.T) {
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		pq   CntAccountHeap
		args args
		want bool
	}{
		// TODO: Add test cases.
		{name: "test for false", pq: newCntAccountHeap,
			args: args{0, 1}, want: false},
		{name: "test for false", pq: newCntAccountHeap,
			args: args{1, 2}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pq.Less(tt.args.i, tt.args.j); got != tt.want {
				t.Errorf("Less() = %v, want %v", got, tt.want)
			}
		})
	}
}


