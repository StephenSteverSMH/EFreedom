package local_http

import "testing"

func TestStart(test *testing.T){
	InitProxyServer("127.0.0.1", 30001);
	HttpProxyServerStart();

}
