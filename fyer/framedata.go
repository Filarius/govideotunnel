package fyer

import "encoding/binary"

type writebuff struct {
	buff []byte
	cap int
	hsize int
}

func (f *writebuff) Init(capacity int, headersize int) {
	f.hsize = 2 + headersize*0
	f.cap = capacity
	f.buff = make([]byte,f.cap)
}

func (f *writebuff) Write(data []byte)  {
	f.buff = append(f.buff,data...)
	f.hsize = len(f.buff)
}

func (f *writebuff) GetFrameCnt() int{
	i := (f.cap - f.hsize) % (len(f.buff))
	return i
}

func (f *writebuff) Pop() []byte {
	l := f.cap - f.hsize
	buff := make([]byte,f.cap,f.cap)
	if len(f.buff) <= l {
		binary.LittleEndian.PutUint16(buff[:2],uint16(len(f.buff)))
		copy(buff[2:],f.buff[:])
		f.buff = make([]byte,f.cap)
	}else if len(f.buff) > l {
		binary.LittleEndian.PutUint16(buff[:2],uint16(l))
		copy(buff[2:],f.buff[:l])
		f.buff = f.buff[l:]
	}
	return buff
}

func (f *writebuff)  UnwrapFrame(frame []byte) []byte{
	l := binary.LittleEndian.Uint16(frame[:2])
	return  frame[2:2+l]
}
