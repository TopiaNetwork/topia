package vm

import (
	"sync"

	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpvmcmm "github.com/TopiaNetwork/topia/vm/common"
	"github.com/TopiaNetwork/topia/vm/native"
	"github.com/TopiaNetwork/topia/vm/tvm"
)

var once sync.Once
var vFactory *VMFactory

type VMFactory struct {
	sync  sync.RWMutex
	vmMap map[tpvmcmm.VMType]VirtualMachine
}

func GetVMFactory() *VMFactory {
	once.Do(func() {
		vFactory = &VMFactory{
			vmMap: make(map[tpvmcmm.VMType]VirtualMachine),
		}

		vFactory.RegisterVM(tpvmcmm.VMType_NATIVE, native.NewNativeVM())
		vFactory.RegisterVM(tpvmcmm.VMType_TVM, tvm.NewTopiaVM())
	})

	return vFactory
}

func (f *VMFactory) SetLogger(level tplogcmm.LogLevel, log tplog.Logger) {
	f.sync.RLock()
	defer f.sync.RUnlock()

	for _, vm := range f.vmMap {
		vm.SetLogger(level, log)
	}
}

func (f *VMFactory) RegisterVM(vmType tpvmcmm.VMType, vm VirtualMachine) {
	f.sync.Lock()
	defer f.sync.Unlock()

	_, ok := f.vmMap[vmType]
	if ok {
		panic("Have registered vm type " + vmType.String())
	}

	f.vmMap[vmType] = vm
}

func (f *VMFactory) GetVM(vmType tpvmcmm.VMType) VirtualMachine {
	f.sync.RLock()
	defer f.sync.RUnlock()

	vm, ok := f.vmMap[vmType]
	if ok {
		return vm
	}

	return nil
}
