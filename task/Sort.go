// automatically generated by the FlatBuffers compiler, do not modify

package task

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type Sort struct {
	_tab flatbuffers.Table
}

func GetRootAsSort(buf []byte, offset flatbuffers.UOffsetT) *Sort {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &Sort{}
	x.Init(buf, n+offset)
	return x
}

func (rcv *Sort) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *Sort) Attr() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *Sort) Uids(obj *UidList, j int) bool {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		x := rcv._tab.Vector(o)
		x += flatbuffers.UOffsetT(j) * 4
		x = rcv._tab.Indirect(x)
		obj.Init(rcv._tab.Bytes, x)
		return true
	}
	return false
}

func (rcv *Sort) UidsLength() int {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.VectorLen(o)
	}
	return 0
}

func SortStart(builder *flatbuffers.Builder) {
	builder.StartObject(2)
}
func SortAddAttr(builder *flatbuffers.Builder, attr flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(attr), 0)
}
func SortAddUids(builder *flatbuffers.Builder, uids flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(1, flatbuffers.UOffsetT(uids), 0)
}
func SortStartUidsVector(builder *flatbuffers.Builder, numElems int) flatbuffers.UOffsetT {
	return builder.StartVector(4, numElems, 4)
}
func SortEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
