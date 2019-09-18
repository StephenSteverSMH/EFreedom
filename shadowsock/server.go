package shadowsock

import (
	"EFreedom/message"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"syscall"
)
// 握手包协议常数
const (
	idType  = 0 // address type index
	idIP0   = 1 // ip address start index
	idDmLen = 1 // domain address length index
	idDm0   = 2 // domain address start index

	typeIPv4 = 1 // type is ipv4 address
	typeDm   = 3 // type is domain address
	typeIPv6 = 4 // type is ipv6 address

	lenIPv4   = net.IPv4len + 2 // ipv4 + 2port
	lenIPv6   = net.IPv6len + 2 // ipv6 + 2port
	lenDmBase = 2               // 1addrLen + 2port, plus addrLen
	lenHmacSha1 = 10
)
// 模块全局日志
var logger *log.Logger

type SSServer struct{
	// 本地监听地址
	addr *net.TCPAddr
}

// 全局http代理服务器实例
var GlobalSSServer SSServer

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
	GlobalSSServer.addr = addr
	// 日志输出到标准输出流
	logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	return nil
}
// 入口函数
func ShadowSockServerStart() error{
	if GlobalSSServer.addr == nil{
		logger.Fatal("ShadowSock服务未初始化")
	}
	tcpListener, err := net.ListenTCP("tcp4", GlobalSSServer.addr)
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
		// 开始处理连接
		pCipher, err := NewCipher("aes-256-cfb", "123456")
		go handleConnection(NewSSConn(conn, pCipher), pool)
	}
	return err
}

// 处理连接
func handleConnection(conn net.Conn, pool message.MessagePool) error{
	// 开启SS握手
	remote_addr_str, err := handShake(conn)
	if err!=nil{
		// 握手失败
		fmt.Println(conn.RemoteAddr().String()+"握手失败")
		return err
	}
	fmt.Println(conn.RemoteAddr().String()+"握手成功")
	remote, err := net.Dial("tcp", remote_addr_str)
	if err != nil {
		fmt.Println("连接不上目标服务器")
		if ne, ok := err.(*net.OpError); ok && (ne.Err == syscall.EMFILE || ne.Err == syscall.ENFILE) {
			// log too many open file error
			// EMFILE is process reaches open file limits, ENFILE is system limit
		} else {
		}
		return err
	}
	fmt.Println("来源"+conn.RemoteAddr().String()+"连接目标服务器成功");
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


// SS握手
func handShake(conn net.Conn) (string, error){
	addr_str, err :=resolveHostPortByShake(conn)
	if err!=nil{
		fmt.Println(err.Error());
		// 解析失败
		return "", err
	}
	return addr_str, nil
}

func resolveHostPortByShake(conn net.Conn) (host string,err error){
	// buf size should at least have the same size with the largest possible
	// request size (when addrType is 3, domain name has at most 256 bytes)
	// 1(addrType) + 1(lenByte) + 255(max length address) + 2(port) + 10(hmac-sha1)
	buf := make([]byte, 269)
	// read till we get possible domain length field
	if _, err = io.ReadFull(conn, buf[:idType+1]); err != nil {
		return
	}

	var reqStart, reqEnd int
	addrType := buf[idType]
	switch addrType & 0xff {
	case typeIPv4:
		reqStart, reqEnd = idIP0, idIP0+lenIPv4
	case typeIPv6:
		reqStart, reqEnd = idIP0, idIP0+lenIPv6
	case typeDm:
		if _, err = io.ReadFull(conn, buf[idType+1:idDmLen+1]); err != nil {
			return
		}
		reqStart, reqEnd = idDm0, idDm0+int(buf[idDmLen])+lenDmBase
	default:
		return
	}
	// 读取地址+端口
	if _, err = io.ReadFull(conn, buf[reqStart:reqEnd]); err != nil {
		return
	}
	// 没有hmac校验

	// Return string for typeIP is not most efficient, but browsers (Chrome,
	// Safari, Firefox) all seems using typeDm exclusively. So this is not a
	// big problem.
	switch addrType & 0xf {
	case typeIPv4:
		host = net.IP(buf[idIP0 : idIP0+net.IPv4len]).String()
	case typeIPv6:
		host = net.IP(buf[idIP0 : idIP0+net.IPv6len]).String()
	case typeDm:
		host = string(buf[idDm0 : idDm0+int(buf[idDmLen])])
	}
	port := binary.BigEndian.Uint16(buf[reqEnd-2 : reqEnd])
	host = net.JoinHostPort(host, strconv.Itoa(int(port)))
	return
}

func Pipe(src, dst net.Conn, buf *message.Message) {
	for {
		// 清空msg.Data
		message.EmptyMessage(buf, message.DefaultMessageSize)
		n, err := src.Read(buf.Data)
		fmt.Print(string("debug的数据:"+string(buf.Data[0:n])))
		fmt.Println("转发数据，"+src.RemoteAddr().String()+"->"+dst.RemoteAddr().String())
		// read may return EOF with n > 0
		// should always process n > 0 bytes before handling error
		if n > 0 {
			// Note: avoid overwrite err returned by Read.
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