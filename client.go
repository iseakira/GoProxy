package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

func main() {
	//ブラウザからHTTP1プロキシるクエストを待ち受け
	listener, err := net.Listen("tcp",":8080")

	if err != nil {
		log.Fatal("起動失敗:",err)
	}

	log.Println("プロキシクライアント起動 :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("接続エラー",err)
			continue
		}
		go handleBrowser(conn)
	}
}

func handleBrowser(browserConn net.Conn) {
	defer browserConn.Close()

	buf := make([]byte,4096)
	n,err := browserConn.Read(buf)
	if err != nil {
		return
	}
	request := string(buf[:n])


	var target string

	if len(request) > 7 && request[:7] == "CONNECT" {
		var method, host, proto string
		fmt.Sscanf(request,"%s %s %s",&method,&host,&proto)
		target = host
	}else {
		for _, line := range strings.Split(request,"\r\n"){
			if strings.HasPrefix(line,"Host:") {
				target = strings.TrimPrefix(line,"Host: ") + ":80"
				break
			}
		}
	}

	if target == "" {
		log.Println("接続先不明")
		return
	}

	log.Println("接続要求:",target)

	//server.goにTLSで接続

	tlsConn, err := tls.Dial("tcp", "localhost:9000", &tls.Config{
		InsecureSkipVerify: true,
	})

	if err != nil {
		log.Println("サーバーへの接続失敗:",err)
		return
	}
	defer tlsConn.Close()

	tlsConn.Write([]byte(target))

	resp := make([]byte, 10)
	tlsConn.Read(resp)
	if string(resp[:2]) != "OK" {
		log.Println("サーバーエラー")
	}

	if len(request) > 7 && request[:7] == "CONNECT" {
		browserConn.Write([]byte("HTTP1.1 200 Connection EStablished\r\nr\n"))
	} else {
		tlsConn.Write(buf[:n])
	}

	go io.Copy(tlsConn,browserConn)
	io.Copy(browserConn,tlsConn)
}