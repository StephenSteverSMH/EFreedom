package main

import (
	"fmt"
	"log"
	"net/url"
	"testing"
)

func TestMM(t *testing.T)  {
	u, err := url.Parse("http://bing.com/search?q=你好#3")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(u.RawPath)
	fmt.Println(u.Query().Encode())


}
