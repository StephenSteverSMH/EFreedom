package local_http

import (
	"EFreedom/message"
	"EFreedom/shadowsock"
	"bufio"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// 模块全局日志
var logger *log.Logger

// 远程代理地址
const remote_addr = "10.28.202.74:15001"

// http代理服务器
type ProxyServer struct{
	// 本地监听地址
	addr *net.TCPAddr
}

// CONNECT Response
const connectRespose = "HTTP/1.1 200 Connection Established\r\n\r\n"

// 全局http代理服务器实例
var GlobalProxyServer ProxyServer

// 初始化服务器
func InitProxyServer(ip string, port uint16)  error{
	ip_str := ip
	port_str := strconv.Itoa(int(port))
	// x.x.x.x:xx
	address_str := ip_str + ":" + port_str
	// 得到TCPAddr
	addr, err := net.ResolveTCPAddr("tcp", address_str)
	if err!=nil{
		return err
	}
	GlobalProxyServer.addr = addr
	// 日志输出到标准输出流
	logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	return nil
}

// 入口函数
func HttpProxyServerStart() error{
	if GlobalProxyServer.addr == nil{
		logger.Fatal("http本地代理未初始化")
	}
	tcpListener, err := net.ListenTCP("tcp4", GlobalProxyServer.addr)
	if err!=nil{
		return err
	}
	// 建立消息池
	pool := message.CreatePool(message.DefaultMessageSize, message.DefaultMessageChanSize)
	for {
		conn, err := tcpListener.Accept()
		if err!=nil{
			break
		}
		pCipher, err := shadowsock.NewCipher("aes-256-cfb", "123456")
		// 开始处理连接
		go handleConnection(shadowsock.NewSSConn(conn, pCipher), pool)
	}
	return err
}

// 处理连接
func handleConnection(conn net.Conn, pool message.MessagePool) error{

	// 开启http代理握手
	remote, err := handShake(conn)
	if remote==nil||err!=nil{
		fmt.Println("远端SS服务器没有开启")
		conn.Close()
		return err
	}
	// 读写缓冲区
	writeBuf := message.GetMessage(&pool)
	readBuf := message.GetMessage(&pool)
	go Pipe(conn, remote,writeBuf)
	Pipe(remote, conn, readBuf)
	// 退出函数
	defer func() {
		if writeBuf!=nil{
			// 清空msg.Data
			message.EmptyMessage(writeBuf, message.DefaultMessageSize)
			// 将msg放回池中
			message.PutMessage(writeBuf, &pool)
		}
		if readBuf!=nil{
			// 清空msg.Data
			message.EmptyMessage(readBuf, message.DefaultMessageSize)
			// 将msg放回池中
			message.PutMessage(readBuf, &pool)
		}
		// 关闭连接
		conn.Close()
		remote.Close()
	}()
	return err
}

// http代理握手
func handShake(conn net.Conn) (net.Conn, error){
	connReader := bufio.NewReader(conn)
	req, err := http.ReadRequest(connReader)
	if err!=nil{
		return nil, err
	}
	fmt.Println(req.Method)
	if req.Method=="CONNECT" {
		// https
		fmt.Println(req.Host)
		conn.Write([]byte(connectRespose))
		remote, err := net.Dial("tcp", remote_addr)
		if err != nil {
			fmt.Println("远端SS节点没有开启")
			if ne, ok := err.(*net.OpError); ok && (ne.Err == syscall.EMFILE || ne.Err == syscall.ENFILE) {
				// log too many open file error
				// EMFILE is process reaches open file limits, ENFILE is system limit
			} else {
			}
			return nil, err
		}
		host :=strings.Split(req.Host,":")
		remote.Write(buildSSPackage(host[0], host[1]))
		//remote.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43})
		return remote, nil
	}
	// 非https
	remote, err := net.Dial("tcp", remote_addr)
	if err != nil {
		if ne, ok := err.(*net.OpError); ok && (ne.Err == syscall.EMFILE || ne.Err == syscall.ENFILE) {
			// log too many open file error

		} else {
		}
		return nil, err
	}
	host :=strings.Split(req.Host,":")
	if len(host)==1{
		// 没有带端口
		host = append(host, "80")
	}
	fmt.Println(conn.RemoteAddr().String()+"握手时http数据包的host和port"+host[0]+host[1])
	ssPackage := buildSSPackage(host[0], host[1])
	fmt.Println("SS包"+hex.EncodeToString(ssPackage))
	remote.Write(ssPackage)
	//没有hmac校验
	//remote.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43})
	req.Write(remote)
	return remote, nil
}

// 建立SS握手包
func buildSSPackage(host, port string) []byte{
	buf := make([]byte, 2 + len(host)+ 2)
	// AtypDomainName
	buf[0] = 0x03
	buf[1] = byte(len(host))
	copy(buf[2:], []byte(host))
	port_u, _ := strconv.ParseUint(port, 10, 16)
	fmt.Println(port_u)
	copy(buf[2+len(host):], []byte{byte(port_u>>8), byte(port_u)})
	return buf
}

func resolveHostPortByShake(conn net.Conn){
	rd := bufio.NewReader(conn)
	reqHead, _ := rd.ReadString('\n')
	reqHead = strings.TrimRight(reqHead, "\r")
	fmt.Println(reqHead)
}

func Pipe(src, dst net.Conn, buf *message.Message) {
	for {
		// 清空msg.Data
		message.EmptyMessage(buf, message.DefaultMessageSize)
		n, err := src.Read(buf.Data)
		if n > 0 {
			// Note: avoid overwrite err returned by Read.
			// 用于调试查看转发的数据
			fmt.Println("转发的数据, "+ src.RemoteAddr().String()+"->"+dst.RemoteAddr().String())
			if _, err := dst.Write(buf.Data[0:n]); err != nil {
				fmt.Println("write error")
				break
			}
		}
		if err != nil {
			// Always "use of closed network connection", but no easy way to
			// identify this specific error. So just leave the error along for now.
			// More info here: https://code.google.com/p/go/issues/detail?id=4373
			/*
				if bool(Debug) && err != io.EOF {
					Debug.Println("read:", err)
				}
			*/
			break
		}
	}
	return
}
