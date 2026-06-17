package main

import (
	"crypto/tls"
	"io"
	"log"
	"net"
)

func main() {
	// TLS証明書を読み込む
	cert, err := tls.LoadX509KeyPair("certs/server.crt", "certs/server.key")
	if err != nil {
		log.Fatal("証明書の読み込み失敗", err)
	}

	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	// TLSで待ち受け開始
	listener, err := tls.Listen("tcp", ":9000", config)
	if err != nil {
		log.Fatal("起動失敗:", err)
	}
	log.Println("VPNサーバー起動 :9000")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("接続エラー：", err)
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(clientConn net.Conn) {
	defer clientConn.Close()

	// クライアントからホスト:ポートを受信
	buf := make([]byte, 256)
	n, err := clientConn.Read(buf)
	if err != nil {
		log.Println("リクエスト読込エラー:", err)
		return
	}

	target := string(buf[:n])
	log.Printf("[サーバー] 接続要求: %s\n", target)

	// 実際のサーバーに接続
	targetConn, err := net.Dial("tcp", target)
	if err != nil {
		log.Println("接続失敗:", target, err)
		clientConn.Write([]byte("ERROR"))
		return
	}
	defer targetConn.Close()

	// 成功応答
	clientConn.Write([]byte("OK"))

	log.Println("[サーバー] データ転送開始")

	// ブラウザ ↔ Webサイト間でバイナリデータを双方向転送
	go io.Copy(targetConn, clientConn)
	io.Copy(clientConn, targetConn)
}