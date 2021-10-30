package fyer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)
//read data from video
type Reader struct {
	frame       Frame
	cfg         Config
	proc        *exec.Cmd
	outpipe		io.ReadCloser
	bitIterator func(byte)byte
	buftemp 	[]byte
}

func (w *Reader) Init(cfg Config) {
	w.cfg = cfg
	w.bitIterator = getBitIterator()
	w.frame.init(&cfg)
	w.buftemp = make([]byte,0)

	var cmdString string

	//cmdString = "streamlink https://www.twitch.tv/esl_csgob best -O | ffmpeg -y -i - -c:v libx264 -crf 25 -preset veryfast -c:a aac 1.mkv"

	cmdString = fmt.Sprintf("-y -i %s -f image2pipe -pix_fmt gray -vcodec rawvideo -",
		cfg.Path,
	)

	cmdFields := strings.Fields(cmdString)
	//exec.Command("C:\\Soft\\streamlink\\bin\\streamlink.exe ",cmdFields...)
	cmd := exec.Command("ffmpeg", cmdFields...)
	cmd.Stderr = os.Stderr

	var err error
	w.proc = cmd
	w.outpipe, err = cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}

	cmd.Start()
}

func (w *Reader) Read(data []byte) (out int,err error) {
	cur := 0
	size := len(data)
	reader := bufio.NewReaderSize(w.outpipe, int(w.cfg.Height*w.cfg.Width))
	for {
		w.frame.init(&w.cfg)
		n, e := io.ReadFull(reader, w.frame.data)
		/*
			TODO does it need to track option where EOF comes in middle of video frame ?
			seems only possible if world is really goes wrong
		*/
		if e != nil {
			if e != io.EOF {
				panic(e)
			}else{
				if n == 0 {
					err = e
					break
				}
			}
		}

		bufbits := w.frame.readBits()
		bufbytes := bitsToBytes(bufbits)
		if len(w.buftemp) > 0 {						// append buffered tail on beginning
			bufbytes = append(w.buftemp, bufbytes...)
			w.buftemp = make([]byte,0)
		}
		cnt := len(bufbytes)
		if cnt > (size - cur){
			cnt = size - cur
		}
		{
			nn := copy(data[cur:], bufbytes[:cnt])
			if nn != cnt{
				panic("!")
			}
		}
		cur += cnt
		if cnt < len(bufbytes){ 		  // store not wrotten tail of data
			w.buftemp = make([]byte,cnt)
			copy(w.buftemp,bufbytes[cnt:])
		}

		if e == io.EOF{
			err  = e
			break
		}
		if cur == len(data){
			break
		}
	}
	out = cur
	return
}

func (w *Reader) Close() error{
	w.outpipe.Close()
	w.proc.Wait()
	return nil
}