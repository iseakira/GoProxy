package main

import (
	"crypto/tls"
	"io"
	"log"
	"net"
)

func main() {
	//TLS証明書を読み込む

	cert, err := tls.LoadX509KeyPair("certs/server.crt","certs/server.key")
	if err != nil {
		log.Fatal("証明書の読み込み失敗",err)
	}

	config := &tls.Config{Certificates:[]tls.Certificate{cert}}

	//tlsで待ち受け開始！

	listener,err := tls.Listen("tcp",":9000",config)

	if err != nil {
		log.Fatal("起動失敗:",err)
	}
	log.Println("VPNサーバー起動 :9000")

	for {
		conn,err := listener.Accept()
		if err != nil {
			log.Println("接続エラー：",err)
			continue
		}
		go handleConn(conn)
	}


}


//
func handleConn(clientConn net.Conn){
	defer clientConn.Close()

	buf := make([]byte,256)
	n,err := clientConn.Read(buf)//clientconnからデータ受け取り
	if err != nil {
		return
	}
	target := string(buf[:n])
	log.Println("接続要求：",target)

	targetConn,err := net.Dial("tcp",target)

	if err != nil {
		log.Println("接続失敗:",target,err)
		clientConn.Write([]byte("ERROR"))
		return
	}
	defer targetConn.Close()

	clientConn.Write([]byte("OK"))

	go io.Copy(targetConn,clientConn)//client→web
	io.Copy(clientConn,targetConn)//web→client
}