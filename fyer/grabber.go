package fyer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// read data from online stream
type Grabber struct {
	frame       Frame
	cfg         Config
	proc        *exec.Cmd
	procYDL		*exec.Cmd
	inpipeFF      io.WriteCloser
	outpipeFF		io.ReadCloser
	outpipeYDL		io.ReadCloser
	bitIterator func(byte)byte
	buftemp 	[]byte
}

func (w *Grabber) Init(cfg Config,url string) {
	w.cfg = cfg
	w.bitIterator = getBitIterator()
	w.frame.init(&cfg)
	w.buftemp = make([]byte,0)

	var cmdString string

	cmdString = fmt.Sprintf("-y -i - -f image2pipe -pix_fmt gray -vcodec rawvideo -")

	cmdFields := strings.Fields(cmdString)
	cmd := exec.Command("ffmpeg", cmdFields...)
	cmd.Stderr = os.Stderr
	var err error
	w.proc = cmd




	cmdString = fmt.Sprintf("%s best -o - ",
		url,
	)
	cmdFields = strings.Fields(cmdString)
	/*
	cmdString = fmt.Sprintf("-y -i - -f image2pipe -pix_fmt gray -vcodec rawvideo -",
		cfg.Path,
	)
	 */
	cmd_ydl := exec.Command("C:\\Soft\\streamlink\\bin\\streamlink.exe ",cmdFields...)
	cmd_ydl.Stderr = os.Stderr
	w.procYDL = cmd_ydl
	/*
	w.outpipeYDL, err = cmd_ydl.StdoutPipe()
	if err != nil {
		panic(err)
	}
	*/
	w.outpipeFF, err = cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	cmd.Stdin,err = cmd_ydl.StdoutPipe()
	if err != nil {
		panic(err)
	}
/*
	w.inpipeFF, err = cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
*/


	cmd_ydl.Start()
	cmd.Start()
	// streamlink TWITCH_URL best -O | ffmpeg -y -i - -c:v libx264 -crf 25 -preset veryfast -c:a aac 1.mkv
}
// fill data array from source
// data size  = width*height /64 /8
func (w *Grabber) Read(data []byte) (out int,err error) {
/*
	train_cho_cho := func(inpipe *io.ReadCloser,outpipe *io.WriteCloser){
		buf := make([]byte,10240)
		for{
			stopflag := false
			n,e := (*inpipe).Read(buf)
			if e != nil{
				if e!=io.EOF{
					fmt.Println(e)
					panic(e)
				}else{
					stopflag = true
				}
			}
			if (n==0) && (e!=io.EOF){
				fmt.Println(e)
				panic(e)
			}
			var nn int
			nn,e = (*outpipe).Write(buf[:n])
			if e != nil{
					fmt.Println(e)
					panic(e)
			}
			if nn != n {
				fmt.Println(nn,n)
				panic("")
			}
			if stopflag {
				break
			}
		}
	}
*/
	cur := 0
	size := len(data)
	readerFF := bufio.NewReaderSize(w.outpipeFF, int(w.cfg.Height*w.cfg.Width))
	//go train_cho_cho(&(w.outpipeYDL), &(w.inpipeFF))
	for {
		bufbytes := make([]byte,0)
		var e error

		if len(w.buftemp) == 0 { //tail data is empty

			w.frame.init(&w.cfg)
			var n int
			n, e = io.ReadFull(readerFF, w.frame.data)
			/*
				TODO does it need to track option where EOF comes in middle of video frame ?
				seems only possible if world is really goes wrong
			*/
			if e != nil {
				if e != io.EOF {
					panic(e)
				} else {
					if n == 0 {
						err = e
						break
					}else{
						panic("non zero N")
					}
				}
			}
			bufbits := w.frame.readBits()
			bufbytes = bitsToBytes(bufbits)
		}


		if len(w.buftemp) > 0 {						// append new data to stored tail data
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

func (w *Grabber) Close() error{
	w.inpipeFF.Close()
	w.procYDL.Wait()
	w.proc.Wait()
	return nil
}