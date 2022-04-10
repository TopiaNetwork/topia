package transactionpool

import (
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/golang/mock/gomock"
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
			want := tt.pq[1]
			tt.pq.Swap(0, 1)
			got := tt.pq[0]
			if !assert.Equal(t, want, got) {
				t.Error("want", want, "got", got)
			}
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
			want := tt.pq.Len() + 1
			tt.pq.Push(&newCntAccountItem)
			got := tt.pq.Len()
			if !assert.Equal(t, want, got) {
				t.Error("want", want, "got", got)
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
	testitem := make(map[uint64]*basic.Transaction, 0)
	testitem[uint64(1)] = Tx1
	type fields struct {
		items map[uint64]*basic.Transaction
		index *nonceHeap
		cache []*basic.Transaction
	}
	type args struct {
		nonce uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *basic.Transaction
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

func Test_transactionPool_AddLocals(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.queue[Category1]))
	assert.Equal(t, 0, len(pool.pendings.pending[Category1]))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	txs := make([]*basic.Transaction, 0)
	txs = append(txs, Tx1)
	txs = append(txs, Tx2)
	pool.AddLocals(txs)
	assert.Equal(t, 1, len(pool.queues.queue[Category1]))
	assert.Equal(t, 2, pool.queues.queue[Category1][From1].txs.Len())
	assert.Equal(t, 0, len(pool.pendings.pending[Category1]))
	assert.Equal(t, 2, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())
	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
}
