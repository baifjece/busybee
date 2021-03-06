package util

import (
	"runtime"

	"github.com/RoaringBitmap/roaring"
	"github.com/fagongzi/log"
)

// MustParseBM parse a bitmap
func MustParseBM(data []byte) *roaring.Bitmap {
	bm := AcquireBitmap()
	MustParseBMTo(data, bm)
	return bm
}

// MustParseBMTo parse a bitmap
func MustParseBMTo(data []byte, bm *roaring.Bitmap) {
	err := bm.UnmarshalBinary(data)
	if err != nil {
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, true)
		log.Fatalf("BUG: parse bm %+v failed with %+v \n %s", data, err, string(buf[:n]))
	}
}

// MustMarshalBM must marshal BM
func MustMarshalBM(bm *roaring.Bitmap) []byte {
	data, err := bm.MarshalBinary()
	if err != nil {
		log.Fatalf("BUG: write bm failed with %+v", err)
	}

	return data
}

// BMAnd bitmap and
func BMAnd(bms ...*roaring.Bitmap) *roaring.Bitmap {
	value := bms[0].Clone()
	for idx, bm := range bms {
		if idx > 0 {
			value.And(bm)
		}
	}

	return value
}

// BMAndInterface bm and using interface{}
func BMAndInterface(bms ...interface{}) *roaring.Bitmap {
	value := bms[0].(*roaring.Bitmap).Clone()
	for idx, bm := range bms {
		if idx > 0 {
			value.And(bm.(*roaring.Bitmap))
		}
	}

	return value
}

// BMOr bitmap or
func BMOr(bms ...*roaring.Bitmap) *roaring.Bitmap {
	value := AcquireBitmap()
	for _, bm := range bms {
		value.Or(bm)
	}

	return value
}

// BMOrInterface bitmap or using interface{}
func BMOrInterface(bms ...interface{}) *roaring.Bitmap {
	value := AcquireBitmap()
	for _, bm := range bms {
		value.Or(bm.(*roaring.Bitmap))
	}

	return value
}

// BMXOr bitmap xor (A union B) - (A and B)
func BMXOr(bms ...*roaring.Bitmap) *roaring.Bitmap {
	value := bms[0].Clone()
	for idx, bm := range bms {
		if idx > 0 {
			value.Xor(bm)
		}
	}

	return value
}

// BMXOrInterface bitmap xor using interface{}
func BMXOrInterface(bms ...interface{}) *roaring.Bitmap {
	value := bms[0].(*roaring.Bitmap).Clone()
	for idx, bm := range bms {
		if idx > 0 {
			value.Xor(bm.(*roaring.Bitmap))
		}
	}

	return value
}

// BMAndnot bitmap andnot A - (A and B)
func BMAndnot(bms ...*roaring.Bitmap) *roaring.Bitmap {
	and := BMAnd(bms...)
	value := bms[0].Clone()
	value.Xor(and)
	return value
}

// BMAndnotInterface bitmap andnot using interface{}
func BMAndnotInterface(bms ...interface{}) *roaring.Bitmap {
	and := BMAndInterface(bms...)
	value := bms[0].(*roaring.Bitmap).Clone()
	value.Xor(and)
	return value
}

// BMMinus bm1 - bm2
func BMMinus(bm1, bm2 *roaring.Bitmap) *roaring.Bitmap {
	v := bm1.Clone()
	v.And(bm2)
	return BMXOr(bm1, v)
}

// BMRemove bm1 - bm2
func BMRemove(bm1, bm2 *roaring.Bitmap) {
	v := bm1.Clone()
	v.And(bm2)
	bm1.Xor(v)
}

// AcquireBitmap create a bitmap
func AcquireBitmap() *roaring.Bitmap {
	return roaring.NewBitmap()
}

// BMAlloc alloc bm
func BMAlloc(new *roaring.Bitmap, shards ...*roaring.Bitmap) {
	old := BMOr(shards...)
	// in new and not in old
	added := BMMinus(new, old)
	// in old not in new
	removed := BMMinus(old, new)

	if removed.GetCardinality() > 0 {
		for _, shard := range shards {
			BMRemove(shard, removed)
		}
	}

	totalAdded := float64(added.GetCardinality())
	if totalAdded > 0 {
		perAdded := make([]float64, len(shards), len(shards))
		avg := float64(new.GetCardinality()) / float64(len(shards))
		for idx, shard := range shards {
			perAdded[idx] = (avg - float64(shard.GetCardinality()))
		}

		lastOp := len(shards) - 1
		op := 0
		itr := added.Iterator()
		for {
			if !itr.HasNext() {
				break
			}

			if perAdded[op] >= 1 || op == lastOp {
				shards[op].Add(itr.Next())
				perAdded[op]--
				if perAdded[op] <= 0 {
					op++
					if op > lastOp {
						op = 0
					}
				}
				continue
			}

			op++
		}
	}
}

// BMSplit split the bitmap
func BMSplit(bm *roaring.Bitmap, maxSize uint64) []*roaring.Bitmap {
	var values []*roaring.Bitmap
	sub := AcquireBitmap()
	values = append(values, sub)

	itr := bm.Iterator()
	c := uint64(0)
	for {
		if !itr.HasNext() {
			break
		}

		if c >= maxSize {
			sub = AcquireBitmap()
			values = append(values, sub)
			c = 0
		}
		value := itr.Next()
		sub.Add(value)
		c++
	}

	return values
}
