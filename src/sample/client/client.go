//Time    : 2020-03-27 13:39
//Author  : Hugh
//File    : client.go
//Descripe:

package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:33111")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	data1,_:=hex.DecodeString("55aa0801001a2a3a4a5a6a94")

	go func(){
		for{

			d:=make([]byte,1024)
			n,err:=conn.Read(d)
			if err != nil {
				panic(err)
			}
			fmt.Printf("received:%02x\n", d[:n])
			fmt.Printf("received:%s\n", string(d[:n]))
		}
	}()
	go func() {
		for{

			_,err=conn.Write(data1)
			if err != nil {
				panic(err)
			}
			time.Sleep(5*time.Second)
		}
	}()
	select {

	}

}
