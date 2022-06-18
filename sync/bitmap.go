package sync

import "sync/atomic"

type IdBitmap struct {
	First       *BitmapItem
	SplitLength uint64
}

type BitmapItem struct {
	ModValues  []uint64
	Length     uint64
	BeginIndex uint64
	Right      *BitmapItem
}

func (bitmap *IdBitmap) IsExist(num uint64) bool {
	if bitmap == nil || bitmap.First == nil {
		return false
	}
	floor, bit := num/64, uint(num%64)
	item := bitmap.First
	for {
		res := item.isExist(floor, bit)
		if res {
			return res
		}
		item = item.Right
		if item == nil {
			break
		}
	}
	return false
}

func (item *BitmapItem) isExist(floor uint64, bit uint) bool {
	if item == nil || len(item.ModValues) == 0 {
		return false
	}

	//floor, bit := num/64, uint(num%64)
	return floor < (uint64(len(item.ModValues))+item.BeginIndex) && (item.ModValues[floor-item.BeginIndex]&(1<<bit)) != 0
}

func (bitmap *IdBitmap) Add(num uint64) bool {
	floor, bit := num/64, uint(num%64)
	isExist := false
	if bitmap.First == nil {
		bitmap.First = new(BitmapItem)
		if bitmap.SplitLength <= 0 {
			bitmap.SplitLength = 100
		}
	}
	modeValLen := uint64(len(bitmap.First.ModValues))
	if modeValLen == 0 {
		isExist = bitmap.First.add(floor, bit)
	} else if floor >= bitmap.First.BeginIndex {
		if floor >= modeValLen+bitmap.First.BeginIndex+bitmap.SplitLength {
			item := bitmap.First
			itemModValLen := modeValLen
			for {
				if item == nil {
					break
				}
				if item.Right == nil {
					item.Right = new(BitmapItem)
					item = item.Right
					break
				}
				if floor >= itemModValLen+item.BeginIndex+bitmap.SplitLength {
					item = item.Right
					itemModValLen = uint64(len(item.ModValues))
					continue
				} else {
					break
				}
			}
			isExist = item.add(floor, bit)
		} else {
			isExist = bitmap.First.add(floor, bit)
		}
	} else if floor < bitmap.First.BeginIndex {
		//lens := len(bitmap.ModValues)
		if floor < bitmap.First.BeginIndex-bitmap.SplitLength {
			item := new(BitmapItem)
			item.Right = bitmap.First
			bitmap.First = item
			isExist = item.add(floor, bit)
		} else {
			isExist = bitmap.First.add(floor, bit)
		}
	}

	return isExist
}

func (item *BitmapItem) add(floor uint64, bit uint) bool {
	isExist := false
	modeValLen := uint64(len(item.ModValues))
	if modeValLen == 0 {
		item.ModValues = append(item.ModValues, 0)
		if floor > 0 {
			item.BeginIndex = floor
		}
		isExist = true
	} else if floor >= modeValLen+item.BeginIndex {
		for floor >= uint64(len(item.ModValues))+item.BeginIndex {
			item.ModValues = append(item.ModValues, 0)
		}
		isExist = true
	} else if floor < item.BeginIndex {
		n := item.BeginIndex - floor
		newValues := make([]uint64, n)
		newValues = append(newValues, item.ModValues...)
		item.ModValues = newValues
		atomic.AddUint64(&item.BeginIndex, -n)
	}

	remainder := uint64(1 << bit)
	if !isExist && item.ModValues[floor-item.BeginIndex]&remainder != 0 {
		return false
	}

	isExist = true
	item.ModValues[floor-item.BeginIndex] |= remainder
	atomic.AddUint64(&item.Length, 1)
	return isExist
}

func (bitmap *IdBitmap) Sub(num uint64) bool {
	floor, bit := num/64, uint(num%64)
	item := bitmap.First
	if item == nil {
		return false
	}

	for {
		res := item.sub(floor, bit)
		if res {
			return res
		}
		item = item.Right
		if item == nil {
			break
		}
	}

	return false
}

func (item *BitmapItem) sub(floor uint64, bit uint) bool {
subCAS:
	if floor >= uint64(len(item.ModValues))+item.BeginIndex {
		return false
	} else if floor < item.BeginIndex {
		return false
	}
	if item.ModValues[floor-item.BeginIndex]&(1<<bit) == 0 {
		return false
	}
	atomic.AddUint64(&item.ModValues[floor-item.BeginIndex], -(1 << bit))
	ok := atomic.CompareAndSwapUint64(&item.Length, item.Length, item.Length-1)
	if !ok {
		for {
			ok = atomic.CompareAndSwapUint64(&item.Length, item.Length, item.Length-1)
			if ok {
				goto subCAS1
			} else {
				goto subCAS
			}

		}
	}
subCAS1:
	return true
}

func (bitmap *IdBitmap) Len() uint64 {
	lens := uint64(0)
	item := bitmap.First
	if item == nil {
		return 0
	}
	for {
		lens += item.Length
		item = item.Right
		if item == nil {
			break
		}
	}
	return lens
}

func (bitmap *IdBitmap) Value() []uint64 {
	arr := []uint64{}
	item := bitmap.First
	if item == nil {
		return arr
	}
	for {
		itemArr := item.value()
		if len(itemArr) > 0 {
			arr = append(arr, itemArr...)
		}
		item = item.Right
		if item == nil {
			break
		}
	}
	return arr
}

func (item *BitmapItem) value() []uint64 {
	arr := make([]uint64, item.Length)
	index := 0
	for i, v := range item.ModValues {
		if v == 0 {
			continue
		}
		uinti := uint(uint64(i) + item.BeginIndex)
		for j := uint(0); j < 64; j++ {
			if v&(1<<j) != 0 {
				arr[index] = uint64(64*uinti + j)
				index++
			}
		}
	}
	return arr
}

func (bitmap *IdBitmap) FirstValue() uint64 {

	item := bitmap.First
	if item == nil {
		return 0
	}
	for {
		itemArr := item.value()
		if len(itemArr) > 0 {
			return itemArr[0]
		}
		item = item.Right
		if item == nil {
			break
		}
	}
	return 0
}
