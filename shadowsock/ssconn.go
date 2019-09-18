package shadowsock

import (
	"io"
	"net"
)

// shadowsocks的Conn配置
type SSConn struct {
	net.Conn
	*Cipher
}
func NewSSConn(c net.Conn, cipher *Cipher) *SSConn {
	return &SSConn{
		Conn:     c,
		Cipher:   cipher}
}

func (c *SSConn) Close() error {
	return c.Conn.Close()
}
func (c *SSConn) Read(b []byte) (n int, err error) {
	// 解密流为null
	if c.dec == nil {
		iv := make([]byte, c.info.ivLen)
		// 读取初始向量
		if _, err = io.ReadFull(c.Conn, iv); err != nil {
			return
		}
		// 初始化解密流
		if err = c.initDecrypt(iv); err != nil {
			return
		}
	}
	// 创建读缓存数据
	cipherData := make([]byte, len(b))
	// 读到加密数据
	n, err = c.Conn.Read(cipherData)
	if n > 0 {
		c.decrypt(b[0:n], cipherData[0:n])
	}
	return
}

func (c *Conn) Write(b []byte) (n int, err error) {
	var iv []byte
	if c.enc == nil {
		// 构建加密器，并随机生成一个初始向量
		iv, err = c.initEncrypt()
		if err != nil {
			return
		}
	}

	// 创建写缓存
	cipherData := make([]byte, len(b) + len(iv))

	if iv != nil {
		// Put initialization vector in buffer, do a single write to send both
		// iv and data.
		copy(cipherData, iv)
	}
	// 先发初始化向量
	c.encrypt(cipherData[len(iv):], b)
	// 写数据
	n, err = c.Conn.Write(cipherData)
	return
}