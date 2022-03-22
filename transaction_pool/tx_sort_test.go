package transactionpool

import (
	"fmt"
	"github.com/TopiaNetwork/topia/transaction"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

var (
	newCntAccountHeap CntAccountHeap
	newCntAccountItem CntAccountItem
)

func init() {
	newCntAccountItem = CntAccountItem{
		"b0001",
		5,
		-1,
	}

	newCntAccountHeap = CntAccountHeap{
		&CntAccountItem{
			"a0001",
			33,
			0,
		},
		&CntAccountItem{
			"a0002",
			11,
			1,
		}, &CntAccountItem{
			"a003",
			22,
			1,
		},
	}

	Tx1 = settransaction(1)
	Tx2 = settransaction(2)
	Tx3 = settransaction(3)
	Tx4 = settransaction(4)

}

func Test_nonceHeap_Len(t *testing.T) {
	tests := []struct {
		name string
		h    nonceHeap
		want int
	}{
		// TODO: Add test cases.
		{name: "test nonceheapLen",
			h: nonceHeap{uint64(10), uint64(13), uint64(9)}, want: 3},
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
		h    nonceHeap
		args args
		want bool
	}{
		// TODO: Add test cases.
		{name: "test nonceheap less", h: nonceHeap{uint64(11), uint64(18)},
			args: args{0, 1}, want: true},
		{name: "test nonceheap less", h: nonceHeap{uint64(18), uint64(10)},
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

func TestCntAccountHeap_Swap(t *testing.T) {
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		pq   CntAccountHeap
		args args
	}{
		// TODO: Add test cases.
		{name: "test for swap", pq: newCntAccountHeap,
			args: args{0, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.pq.Swap(0, 1)
			fmt.Printf("%v,%v", tt.pq[0], tt.pq[1])
		})
	}
}

func TestCntAccountHeap_Push(t *testing.T) {
	type args struct {
		x interface{}
	}
	tests := []struct {
		name string
		pq   CntAccountHeap
		args args
	}{
		// TODO: Add test cases.
		{name: "heap_push",
			pq:   newCntAccountHeap,
			args: args{newCntAccountItem}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.pq.Push(&newCntAccountItem)
			fmt.Printf("%v", tt.pq.Len())
			for k, _ := range tt.pq {
				fmt.Printf("%v", tt.pq[k])
			}
		})
	}
}

func TestCntAccountHeap_Pop(t *testing.T) {
	tests := []struct {
		name string
		pq   CntAccountHeap
		want interface{}
	}{
		// TODO: Add test cases.
		{name: "pop test",
			pq: newCntAccountHeap,
			want: &CntAccountItem{"a003",
				22,
				-1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pq.Pop(); !reflect.DeepEqual(got, tt.want) {

				t.Errorf("Pop() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txSortedMap_Get(t *testing.T) {
	testitem := make(map[uint64]*transaction.Transaction, 0)
	testitem[uint64(1)] = Tx1
	type fields struct {
		items map[uint64]*transaction.Transaction
		index *nonceHeap
		cache []*transaction.Transaction
	}
	type args struct {
		nonce uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *transaction.Transaction
	}{
		// TODO: Add test cases.
		{"test get", fields{
			items: testitem,
			index: &nonceHeap{uint64(1)},
			cache: nil,
		}, args{uint64(1)}, Tx1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &txSortedMap{
				items: tt.fields.items,
				index: tt.fields.index,
				cache: tt.fields.cache,
			}
			if got := m.Get(tt.args.nonce); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}
