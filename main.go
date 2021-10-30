package main

import (
	"context"
	"fmt"
	"github.com/go-zeromq/zmq4"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

func main(){
	var wg sync.WaitGroup
	wg.Add(1)
	z := time.Now().Unix()
	z+= 1
	cntx := context.Background()
	p1 := zmq4.NewDealer(cntx)
	p2 := zmq4.NewDealer(cntx)

	p1.Listen("tcp://localhost:12345")
	p2.Dial("tcp://localhost:12345")

	go func() {
		r := make([]byte,1000000)
		for{
			p2.Send(zmq4.NewMsg(r))
		}
	}()
	go func() {
		var i int
		t :=time.Now().Add(time.Second)
		for{
			r,e:= p1.Recv()
			if e!=nil{
				fmt.Println("err ",e.Error())
			}else {
				d := r.Bytes()
				if len(d)==0 {
				//	fmt.Print(len(d))
					continue
				}
				d[0]=0
				i+=1
				if t.Before(time.Now()){
					fmt.Println("t ", i)
					i = 0
					t = time.Now().Add(time.Second)
				}
			}

		}
	}()
	wg.Wait()
}

func wormholeсlient(local string,remote string) {
	l, err := net.Listen("tcp", local)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	for {
		// Wait for a connection.
		c, err := l.Accept()
		if err != nil {
			log.Println("Error establshing incoming connection:", err)
			continue
		}
		log.Println("Client connected from", c.RemoteAddr())

		// handle the connection in a goroutine

		go func(c net.Conn){
			defer c.Close()
			log.Println("Opening wormhole from", c.RemoteAddr())
			start := time.Now()

			// connect to the destination tcp port
			destConn, err := net.Dial("tcp", remote)
			if err != nil {
				log.Println("Error connecting to destination port:", err)
				return
			}
			defer destConn.Close()
			log.Println("Wormhole open from", c.RemoteAddr())

			go func() { io.Copy(c, destConn) }()
			io.Copy(destConn, c)

			end := time.Now()
			duration := end.Sub(start)
			log.Println("Closing wormhole from", c.RemoteAddr(), "after", duration)
		}()
	}




}