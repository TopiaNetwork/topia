package transactionpool

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/TopiaNetwork/topia/codec"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

var (
	newSizeAccountHeap SizeAccountHeap
	newSizeAccountItem SizeAccountItem
)

func init() {
	newSizeAccountItem = SizeAccountItem{
		"b0001",
		5,
		-1,
	}

	newSizeAccountHeap = SizeAccountHeap{
		&SizeAccountItem{
			"a0001",
			33333,
			0,
		},
		&SizeAccountItem{
			"a0002",
			11111,
			1,
		}, &SizeAccountItem{
			"a003",
			22222,
			1,
		},
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

func TestSizeAccountHeap_Len(t *testing.T) {
	tests := []struct {
		name string
		pq   SizeAccountHeap
		want int
	}{
		// TODO: Add test cases.
		{name: "test for CntAccountHeap_Len",
			pq:   newSizeAccountHeap,
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

func TestSizeAccountHeap_Less(t *testing.T) {
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		pq   SizeAccountHeap
		args args
		want bool
	}{
		// TODO: Add test cases.
		{name: "test for false", pq: newSizeAccountHeap,
			args: args{0, 1}, want: false},
		{name: "test for false", pq: newSizeAccountHeap,
			args: args{1, 2}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pq.Less(tt.args.i, tt.args.j); got != tt.want {
				t.Errorf("Less() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSizeAccountHeap_Swap(t *testing.T) {
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		pq   SizeAccountHeap
		args args
	}{
		// TODO: Add test cases.
		{name: "test for swap", pq: newSizeAccountHeap,
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

func TestSizeAccountHeap_Push(t *testing.T) {
	type args struct {
		x interface{}
	}
	tests := []struct {
		name string
		pq   SizeAccountHeap
		args args
	}{
		// TODO: Add test cases.
		{name: "heap_push",
			pq:   newSizeAccountHeap,
			args: args{newSizeAccountItem}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := tt.pq.Len() + 1
			tt.pq.Push(&newSizeAccountItem)
			got := tt.pq.Len()
			if !assert.Equal(t, want, got) {
				t.Error("want", want, "got", got)
			}
		})
	}
}

func TestSizeAccountHeap_Pop(t *testing.T) {
	tests := []struct {
		name string
		pq   SizeAccountHeap
		want interface{}
	}{
		// TODO: Add test cases.
		{name: "pop test",
			pq: newSizeAccountHeap,
			want: &SizeAccountItem{"a003",
				22222,
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
	testitem := make(map[uint64]*txbasic.Transaction, 0)
	testitem[uint64(1)] = Tx1
	type fields struct {
		items map[uint64]*txbasic.Transaction
		index *nonceHp
		cache []*txbasic.Transaction
	}
	type args struct {
		nonce uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *txbasic.Transaction
	}{
		// TODO: Add test cases.
		{"test get", fields{
			items: testitem,
			index: &nonceHp{uint64(1)},
			cache: nil,
		}, args{uint64(1)}, Tx1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &txSortedMapByNonce{
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
	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	blockService := NewMockBlockService(ctrl)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	txs := make([]*txbasic.Transaction, 0)
	txs = append(txs, Tx1)
	txs = append(txs, Tx2)
	pool.AddLocals(txs)
	assert.Equal(t, 1, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.queues.getTxListByAddrOfCategory(Category1, From1).txs.Len())
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())
	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
}
