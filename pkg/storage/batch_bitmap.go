package storage

import (
	"bytes"

	"github.com/RoaringBitmap/roaring"
	"github.com/deepfabric/beehive/pb/raftcmdpb"
	bhstorage "github.com/deepfabric/beehive/storage"
	"github.com/deepfabric/busybee/pkg/pb/rpcpb"
	"github.com/deepfabric/busybee/pkg/util"
	"github.com/fagongzi/goetty"
	"github.com/fagongzi/log"
	"github.com/fagongzi/util/protoc"
)

type bitmapBatch struct {
	buf           *goetty.ByteBuf
	bitmaps       [][]byte
	bitmapAdds    []*roaring.Bitmap
	bitmapRemoves []*roaring.Bitmap
	ops           [][]int
}

func newBitmapBatch() batchType {
	return &bitmapBatch{
		buf: goetty.NewByteBuf(256),
	}
}

func (rb *bitmapBatch) support() []rpcpb.Type {
	return []rpcpb.Type{rpcpb.BMCreate, rpcpb.BMAdd, rpcpb.BMRemove, rpcpb.BMClear}
}

func (rb *bitmapBatch) addReq(req *raftcmdpb.Request, resp *raftcmdpb.Response, b *batch, attrs map[string]interface{}) {
	switch rpcpb.Type(req.CustemType) {
	case rpcpb.BMCreate:
		msg := getBMCreateRequest(attrs)
		protoc.MustUnmarshal(msg, req.Cmd)

		msg.Key = req.Key

		rb.add(req.Key, msg.Mod, msg.Value...)

		resp.Value = rpcpb.EmptyRespBytes
	case rpcpb.BMAdd:
		msg := getBMAddRequest(attrs)
		protoc.MustUnmarshal(msg, req.Cmd)

		msg.Key = req.Key
		rb.add(msg.Key, msg.Mod, msg.Value...)

		resp.Value = rpcpb.EmptyRespBytes
	case rpcpb.BMRemove:
		msg := getBMRemoveRequest(attrs)
		protoc.MustUnmarshal(msg, req.Cmd)

		msg.Key = req.Key
		rb.remove(msg.Key, msg.Value...)

		resp.Value = rpcpb.EmptyRespBytes
	case rpcpb.BMClear:
		msg := getBMClearRequest(attrs)
		protoc.MustUnmarshal(msg, req.Cmd)

		msg.Key = req.Key
		rb.clear(msg.Key)

		resp.Value = rpcpb.EmptyRespBytes
	default:
		log.Fatalf("BUG: not supoprt rpctype: %d", rpcpb.Type(req.CustemType))
	}
}

func (rb *bitmapBatch) add(bm []byte, mod uint32, values ...uint32) {
	for idx, key := range rb.bitmaps {
		if bytes.Compare(key, bm) == 0 {
			rb.ops[idx] = append(rb.ops[idx], opAdd)
			rb.appendAdds(idx, mod, values...)
			return
		}
	}

	value := util.AcquireBitmap()
	rb.doAdd(value, mod, values...)

	rb.ops = append(rb.ops, []int{opAdd})
	rb.bitmaps = append(rb.bitmaps, bm)
	rb.bitmapAdds = append(rb.bitmapAdds, value)
	rb.bitmapRemoves = append(rb.bitmapRemoves, nil)
}

func (rb *bitmapBatch) remove(bm []byte, values ...uint32) {
	for idx, key := range rb.bitmaps {
		if bytes.Compare(key, bm) == 0 {
			rb.ops[idx] = append(rb.ops[idx], opRemove)
			rb.appendRemoves(idx, values...)
			return
		}
	}

	value := util.AcquireBitmap()
	value.AddMany(values)

	rb.ops = append(rb.ops, []int{opRemove})
	rb.bitmaps = append(rb.bitmaps, bm)
	rb.bitmapAdds = append(rb.bitmapAdds, nil)
	rb.bitmapRemoves = append(rb.bitmapRemoves, value)
}

func (rb *bitmapBatch) clear(bm []byte) {
	rb.clean(bm, opClear)
}

func (rb *bitmapBatch) del(bm []byte) {
	rb.clean(bm, opDel)
}

func (rb *bitmapBatch) appendAdds(idx int, mod uint32, values ...uint32) {
	if rb.bitmapAdds[idx] == nil {
		rb.bitmapAdds[idx] = util.AcquireBitmap()
	}

	rb.doAdd(rb.bitmapAdds[idx], mod, values...)
}

func (rb *bitmapBatch) doAdd(bm *roaring.Bitmap, mod uint32, values ...uint32) {
	if mod == 0 {
		bm.AddMany(values)
		return
	}

	end := values[1]
	for i := values[0]; i <= end; i++ {
		if i%mod == 0 {
			bm.Add(i)
		}
	}
}

func (rb *bitmapBatch) appendRemoves(idx int, values ...uint32) {
	if rb.bitmapRemoves[idx] == nil {
		rb.bitmapRemoves[idx] = util.AcquireBitmap()
	}

	rb.bitmapRemoves[idx].AddMany(values)
}

func (rb *bitmapBatch) clean(bm []byte, op int) {
	for idx, key := range rb.bitmaps {
		if bytes.Compare(key, bm) == 0 {
			rb.ops[idx] = append(rb.ops[idx], op)

			if rb.bitmapAdds[idx] != nil {
				rb.bitmapAdds[idx] = nil
			}

			if rb.bitmapRemoves[idx] != nil {
				rb.bitmapRemoves[idx] = nil
			}
			return
		}
	}

	rb.ops = append(rb.ops, []int{op})
	rb.bitmaps = append(rb.bitmaps, bm)
	rb.bitmapAdds = append(rb.bitmapAdds, nil)
	rb.bitmapRemoves = append(rb.bitmapRemoves, nil)
}

func (rb *bitmapBatch) reset() {
	rb.buf.Clear()
	rb.ops = rb.ops[:0]
	rb.bitmaps = rb.bitmaps[:0]
	rb.bitmapAdds = rb.bitmapAdds[:0]
	rb.bitmapRemoves = rb.bitmapRemoves[:0]
}

func (rb *bitmapBatch) exec(s bhstorage.DataStorage, b *batch) error {
	if len(rb.ops) > 0 {
		bm := util.AcquireBitmap()
		for idx, ops := range rb.ops {
			key := rb.bitmaps[idx]
			if ops[len(ops)-1] == opDel {
				b.wb.Delete(key)
				b.changedBytes -= int64(len(key))
				continue
			}

			value, err := s.Get(key)
			if err != nil {
				return err
			}

			if len(value) > 0 {
				bm = util.AcquireBitmap()
				util.MustParseBMTo(value, bm)
			}

			for _, op := range ops {
				switch op {
				case opAdd:
					bm = util.BMOr(bm, rb.bitmapAdds[idx])
					break
				case opRemove:
					bm = util.BMXOr(bm, rb.bitmapRemoves[idx])
					break
				case opClear:
					bm = util.AcquireBitmap()
					break
				case opDel:
					bm = util.AcquireBitmap()
					break
				}
			}

			data := util.MustMarshalBM(bm)
			b.wb.Set(key, util.MustMarshalBM(bm))

			b.writtenBytes += uint64(len(data) - len(value))
			b.changedBytes += int64(len(data) - len(value))
		}
	}

	return nil
}
