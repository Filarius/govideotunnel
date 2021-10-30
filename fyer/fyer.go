package fyer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"

	//"math/rand"
	///"os"
	//"os/exec"
	"strings"
)

type Frame struct {
	data []byte
	curx int
	cury int
	w    int
	h    int
}

type Config struct {
	Width,Height uint32
	Path       string
	IsDecoding bool
}



func main8() {
	/*
	cfgw := Config{
		width:      512 * 2,
		height:     512 * 2,
		path:       "ffmpeg.zip",
		pathOut:    "v.flv",
		flagDecode: false,
	}

	Write(cfgw)

	cfgr := Config{
		width:      512 * 2,
		height:     512 * 2,
		path:       "v.flv",
		pathOut:    "ffmpeg2.zip",
		flagDecode: false,
	}

	Read(cfgr)
	*/
}

func Write(cfg Config) {
	cmdString := fmt.Sprintf("-y -f rawvideo -vcodec rawvideo -s %dx%d -pix_fmt gray -r 60 -i - -c:v libx264 -preset veryfast -crf 24 %s",
		cfg.Width,
		cfg.Height,
		cfg.Path,
	)
	cmdFields := strings.Fields(cmdString)
	cmd := exec.Command("ffmpeg", cmdFields...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	inp, err := cmd.StdinPipe()
	if err != nil {
		fmt.Print(err)
		return
	}
	cmd.Start()

	file, err := os.Open(cfg.Path)
	{
		buf := make([]byte, cfg.Width*cfg.Height/(64*8))
		bufbits := make([]byte, len(buf)*8)
		bitIterator := getBitIterator()
		for {
			n, e := file.Read(buf)
			if e != nil {
				break
			}
			for i := 0; i < n; i++ {
				for j := 0; j < 8; j++ {
					bufbits[i*8+j] = bitIterator(buf[i])
				}
			}
			var frame Frame
			frame.init(&cfg)
			frame.writeBits(bufbits)
			//fmt.Print(frame.isFull())
			inp.Write(frame.data)
		}
	}
	inp.Close()
	file.Close()
	cmd.Wait()
}

func Read(cfg Config) {
	cmdString := fmt.Sprintf("-y -i %s -f image2pipe -pix_fmt gray -vcodec rawvideo -",
		cfg.Path,
	)
	cmdFields := strings.Fields(cmdString)
	cmd := exec.Command("ffmpeg", cmdFields...)
	cmd.Stderr = os.Stderr
	op, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Print(err)
		return
	}

	reader := bufio.NewReaderSize(op, int(cfg.Height*cfg.Width))

	cmd.Start()

	file, err := os.Create(cfg.Path)
	{
		for {
			var frame Frame
			frame.init(&cfg)
			//bb :=make([]byte,len(frame.data))
			n, e := io.ReadFull(reader, frame.data)

			if e != nil {
				if e != io.EOF {
					panic(e)
				}
			}
			if n == 0 {
				fmt.Print("read end")
				break
			}
			if n != len(frame.data) {
				panic("!")
			}

			bufbits := frame.readBits()
			buf := bitsToBytes(bufbits)

			_, e = file.Write(buf)
			if e != nil {
				panic(e)
			}
		}
	}

	op.Close()
	file.Close()

	cmd.Wait()
}



func (f *Frame) init(cfg *Config) {

	if (cfg.Height % 8) != 0 {
		panic("Height must be divider of 8!")
	}
	if (cfg.Width % 8) != 0 {
		panic("Width must be divider of 8!")
	}
	f.w = int(cfg.Width)
	f.h = int(cfg.Height)
	f.curx = 0
	f.cury = 0
	f.data = make([]byte, uint32(cfg.Height)*uint32(cfg.Width))
}

//input is array of bits (zeros and ones)
//return count of written bits
func (f *Frame) writeBits(b []byte) int {
	l := f.Len()
	cap := f.Capacity()
	if l == cap {
		return 0
	}
	cnt := len(b)
	if cnt > (cap - l){
		cnt = cap - l
	}

	for i:=0;i<cnt;i++{
		f.atomicWriteBit(b[i])
	}

	return cnt
}

func (f *Frame) readBits() []byte {
	mx := f.w / 8
	my := f.h / 8
	buf := make([]byte, mx*my)
	for i:=0;i<len(buf);i++{
		buf[i] = f.atomicReadBit()
	}
	return buf
}


func (f *Frame) atomicReadBit() byte{
	//watch for bound in another place
	//fmt.Printf("%d %d \n",f.cury,f.curx)
	var sum uint16 = 0
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			sum += uint16(f.data[(f.cury*8 + y)*f.w + (f.curx*8 + x)])
		}
	}
	if sum > (127 * 64) {
		sum = 1
	} else {
		sum = 0
	}

	f.curx++
	if f.curx == (f.w / 8) {
		f.curx = 0
		f.cury++
	}
	return byte(sum)
}

func (f *Frame) atomicWriteBit(bit byte) {
	//watch for bound in another place
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			f.data[(f.cury*8 + y)*f.w + (f.curx*8 + x)] = 255 * bit
		}
	}
	f.curx++
	if f.curx == (f.w / 8) {
		f.curx = 0
		f.cury++
	}
}

func (f *Frame) Len() int {
	return f.cury*(f.w/8) + f.curx
}

func (f *Frame) Capacity() int {
	return len(f.data) / 64
}

func bitsToBytes(b []byte) []byte {
	if (len(b) % 8) != 0 {
		panic("array size must be divider of 8 !")
	}
	buf := make([]byte, len(b)/8)
	for i := 0; i < len(buf); i++ {
		a := byte(0)
		for j := 7; j >= 0; j-- {
			a <<= 1
			a |= b[i*8+(7-j)]

		}
		buf[i] = a
	}
	return buf
}

func (f *Frame) isFull() bool {
	return f.cury == (f.h / 8)
}

func getBitIterator() func(byte) byte {
	var mask uint8 = 1
	var i uint8 = 0
	return func(b byte) byte {
		for {
			if mask == 1 {
				mask = 1 << 7
				i = 7
			} else {
				mask = mask >> 1
				i--
			}
			return (b & mask) >> i
		}
	}
}
