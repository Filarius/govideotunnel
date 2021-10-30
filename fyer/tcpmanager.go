package fyer

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Tcpmanager struct{
	grabber Grabber
	reader Reader
	writer Writer
	wbuff *writebuff
	wbrw sync.Mutex
}


func (t *Tcpmanager) Start(
	localaddr string,
	remoteaddr string,
	inputurl string,
	outputurl string,
	blocksize int) {


	//Tcpmanager modes
	// represent TCP R/W and Stream R/W
	fsr := false
	fsw := false
	ftr := false
	ftw := false


	if localaddr == ""{
		panic("Please define local TCP address")
	}
	if (inputurl=="")&&(outputurl==""){
		panic("Please define input or output video stream")
	}
	if (inputurl=="")||(outputurl==""){
		if (remoteaddr=="") {
			panic("Please define remote TCP address")
		}
	}

	if outputurl != ""{
		ftw  = true
		fsr = true
	}
	if inputurl != ""{
		ftr = true
		fsw = true
	}
	if ftr && ftw {
		fsw = false
		fsr = false
	}


	t.wbuff.Init(blocksize, 2)

	l, err := net.Listen("tcp", localaddr)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	for {
		localConn, err := l.Accept()
		if err != nil {
			log.Println("Error establshing incoming connection:", err)
			continue
		}
		if ftr||ftw {
			destConn, err := net.Dial("tcp", remoteaddr)
			if err != nil {
				log.Println("Error connecting to destination port:", err)
				return
			}
			if ftr {
				go func(){
					io.Copy(localConn,destConn)
				}()
			}
			if ftw {
				go func(){
					io.Copy(destConn,localConn)
				}()
			}
		}
		if fsr {
			cfg := Config{
				Width:      1280,
				Height:     720,
				Path:       inputurl,
				IsDecoding: false,
			}
			t.grabber.Init(cfg,inputurl)

			go func(){
				defer t.grabber.Close()
				fdata := make([]byte,cfg.Height*cfg.Width/(64*8))
				for{
					n,e := t.grabber.Read(fdata)
					if n != len(fdata){
						fmt.Println("frame readed not fully")
						break
					}
					if e != nil{
						fmt.Println("frame read error")
						fmt.Println(e.Error())
						break
					}
					fdata = t.wbuff.UnwrapFrame(fdata)
					for i:=0;i<len(fdata);	{
						k,ee := localConn.Write(fdata[i:])
						if ee != nil{
							fmt.Println("conn write error")
							fmt.Println(ee.Error())
							panic("")
						}
						if k == 0 {
							fmt.Println("conn zero wrotten error")
							panic("")
						}
						i += k
					}
				}

			}()
		}
		if fsw {
			cfg := Config{
				Width:      1280,
				Height:     720,
				Path:       outputurl,
				IsDecoding: false,
			}
			go func() { // async read from TCP
				for {

					if t.wbuff.GetFrameCnt() == 0 {
						tmp := make([]byte, 10000)
						n,e := localConn.Read(tmp)
						if e != nil{
							fmt.Println("local tcp read error")
							fmt.Println(e.Error())
							break
						}
						if n>0 {
							t.wbrw.Lock()
							t.wbuff.Write(tmp[:n])
							t.wbrw.Unlock()
						}
					}else{
						time.Sleep(time.Second/(60*2))
					}
				}
			}()
			go func() { // async write to video writer
				t.writer.Init(cfg)
				updatetimeout := time.Now()
				for{
					if updatetimeout.Before(time.Now()) {
						updatetimeout.Add(time.Second/60)
						t.writer.Write(t.wbuff.Pop())
					}
				}

				}()


			}


	}
}