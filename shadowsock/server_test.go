package shadowsock

import "testing"

func TestSSServer(t *testing.T){
	InitProxyServer("127.0.0.1", 15001)
	HttpProxyServerStart()
}
