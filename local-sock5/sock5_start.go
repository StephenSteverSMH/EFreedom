package local_sock5

import (
	"fmt"
	"log"
	"net"
	"os"
)

var logger *log.Logger

const (
	PackageSize = 512
)

func Sock5ServerStart(){
	if(logger==nil){
		logger = log.New(os.Stdout, "", log.Ldate)
	}
	tcpAddr, err:= net.ResolveTCPAddr("tcp", "127.0.0.1:30001");
	if err!=nil{
		return
	}
	// 先监听
	tcpListener, err := net.ListenTCP("tcp4", tcpAddr)
	for(true){
		conn, err := tcpListener.Accept()
		if(err!=nil){
			fmt.Println(err.Error())
			return
		}
		fmt.Println("接收到一个连接"+conn.RemoteAddr().String())
		readBuffer := make([]byte, PackageSize)
		n, err :=conn.Read(readBuffer)
		if(err!=nil){
			fmt.Println(err.Error())
			return
		} else if(n==0){

		}
		fmt.Println(string(readBuffer))
		conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		//readBuffer = readBuffer[0:0]
		n, err = conn.Read(readBuffer)
		if err!=nil{
			fmt.Println(err.Error())
			continue;
		}
		fmt.Println(string(readBuffer))
	}
}

