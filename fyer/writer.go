package fyer

import (
	//"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

//write data to videpo
type Writer struct {
	frame       Frame
	cfg         Config
	proc        *exec.Cmd
	procYDL		*exec.Cmd
	inpipe      io.WriteCloser
	outpipe		io.ReadCloser
	bitIterator func(byte)byte
	buftemp 	[]byte
}

func (w *Writer) Init(cfg Config) {
	w.cfg = cfg
	w.bitIterator = getBitIterator()
	w.frame.init(&cfg)
	w.buftemp = make([]byte,0)

	var cmdString string
		cmdString = fmt.Sprintf("-y -f  rawvideo -vcodec rawvideo -r 100 -s %dx%d -pix_fmt gray -re -i -  -c:v libx264 -preset veryfast -crf 20 -g 20 -f flv %s",
			cfg.Width,
			cfg.Height,
			cfg.Path,
		)

	cmdFields := strings.Fields(cmdString)
	cmd := exec.Command("ffmpeg", cmdFields...)
	cmd.Stderr = os.Stderr

	var err error
		cmd.Stdout = os.Stdout
		w.proc = cmd
		w.inpipe, err = cmd.StdinPipe()
		if err != nil {
			panic(err)
		}

/*
	cmdString = fmt.Sprintf("%s -o - ",url)
	cmdFields = strings.Fields(cmdString)
	cmd_ydl := exec.Command("youtubedl.exe",cmdFields...)
	w.procYDL = cmd_ydl
*/
	cmd.Start()
}

func (w *Writer) Write(data []byte) (int, error) {
	bitbuf := make([]byte,len(data)*8)
	for i:=0;i<len(data);i++{
		for j:=0;j<8;j++{
			bitbuf[i*8+j] = w.bitIterator(data[i])
		}
	}
	cur := 0
	for{
		n := w.frame.writeBits(bitbuf[cur:])
		if n == 0 {
			panic("!")
		}
		cur += n
		if w.frame.isFull(){
			w.inpipe.Write(w.frame.data)
			w.frame.init(&w.cfg)
		}
		if cur == len(bitbuf){
			break
		}
	}
	return len(data),nil
}

func (w *Writer) Close() error{
		w.inpipe.Close()

	w.proc.Wait()
	return nil
}
