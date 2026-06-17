package main

import (
	"bufio"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

var serverAddr string

func main() {
	// サーバーアドレスを環境変数から取得（デフォルト: localhost:9000）
	serverAddr = os.Getenv("VPN_SERVER")
	if serverAddr == "" {
		serverAddr = "localhost:9000"
	}

	log.Printf("VPNサーバー: %s\n", serverAddr)

	// HTTPプロキシをリッスン
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("起動失敗:", err)
	}

	log.Println("VPNクライアント起動 :8080 (HTTPプロキシ)")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("接続エラー", err)
			continue
		}
		go handleHTTPProxy(conn)
	}
}

func handleHTTPProxy(clientConn net.Conn) {
	defer clientConn.Close()

	reader := bufio.NewReader(clientConn)

	// HTTPリクエストを読み込む
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("リクエスト解析エラー:", err)
		return
	}

	// ホスト名とポート番号を取得
	host := req.Host
	if !strings.Contains(host, ":") {
		if req.Method == "CONNECT" {
			host = host + ":443"
		} else {
			host = host + ":80"
		}
	}

	log.Printf("[クライアント] 接続要求: %s %s\n", req.Method, host)

	// サーバーにTLSで接続
	tlsConn, err := tls.Dial("tcp", serverAddr, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Println("サーバー接続失敗:", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer tlsConn.Close()

	// サーバーに接続先を送信
	tlsConn.Write([]byte(host))

	// サーバーからの応答待機
	respBuf := make([]byte, 10)
	n, err := tlsConn.Read(respBuf)
	if err != nil || n < 2 || string(respBuf[:2]) != "OK" {
		log.Println("サーバーエラー")
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	log.Println("[クライアント] サーバー接続成功")

	// CONNECTメソッドの場合
	if req.Method == "CONNECT" {
		// 200 Connection Establishedを返す
		clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		log.Println("[クライアント] CONNECT トンネル確立")
	} else {
		// 通常のHTTPリクエストをサーバーに転送
		req.RequestURI = ""
		req.Write(tlsConn)
	}

	log.Println("[クライアント] データ転送開始")

	// ブラウザとサーバー間でバイナリデータを双方向転送
	go io.Copy(tlsConn, clientConn)
	io.Copy(clientConn, tlsConn)
}