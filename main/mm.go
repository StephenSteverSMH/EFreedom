package main

import (
	"EFreedom/local-http"
	"fmt"
)

func main(){
	local_http.InitProxyServer("127.0.0.1", 30001)
	err:=local_http.HttpProxyServerStart()
	fmt.Println(err.Error())
}
