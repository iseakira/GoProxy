package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

type ProxyRequest struct {
	Method  string            `json:"method"`
	Host    string            `json:"host"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type ProxyResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func main() {
	//ブラウザからHTTPプロキシリクエストを待ち受け
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

	reader := bufio.NewReader(browserConn)
	httpReq, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("HTTPリクエスト解析エラー:", err)
		return
	}

	// リクエスト情報を抽出
	host := httpReq.Host
	if !strings.Contains(host, ":") {
		if httpReq.URL.Scheme == "https" {
			host = host + ":443"
		} else {
			host = host + ":80"
		}
	}

	// ボディを読む
	body := ""
	if httpReq.Body != nil {
		bodyBytes, err := io.ReadAll(httpReq.Body)
		if err == nil {
			body = string(bodyBytes)
		}
	}

	// ヘッダを辞書に変換（Host除外）
	headers := make(map[string]string)
	for key, values := range httpReq.Header {
		if key != "Host" && len(values) > 0 {
			headers[key] = values[0]
		}
	}

	proxyReq := ProxyRequest{
		Method:  httpReq.Method,
		Host:    strings.Split(host, ":")[0], // ホスト名のみ
		Path:    httpReq.URL.RequestURI(),
		Headers: headers,
		Body:    body,
	}

	log.Println("接続要求:", proxyReq.Method, proxyReq.Host, proxyReq.Path)

	// サーバーにTLSで接続
	tlsConn, err := tls.Dial("tcp", "localhost:9000", &tls.Config{
		InsecureSkipVerify: true,
	})

	if err != nil {
		log.Println("サーバーへの接続失敗:",err)
		browserConn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
		return
	}
	defer tlsConn.Close()

	// JSONでリクエストを送信
	encoder := json.NewEncoder(tlsConn)
	log.Println("[クライアント] サーバーへリクエスト送信:")
	log.Printf("  Method: %s, Host: %s, Path: %s\n", proxyReq.Method, proxyReq.Host, proxyReq.Path)
	if err := encoder.Encode(&proxyReq); err != nil {
		log.Println("リクエスト送信エラー:", err)
		return
	}

	// レスポンスを受信
	decoder := json.NewDecoder(tlsConn)
	var proxyResp ProxyResponse
	if err := decoder.Decode(&proxyResp); err != nil {
		log.Println("レスポンス受信エラー:", err)
		browserConn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
		return
	}

	log.Printf("[クライアント] サーバーからレスポンス受信: ステータス %d\n", proxyResp.Status)

	// HTTPレスポンスをブラウザに送信
	respLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", proxyResp.Status, http.StatusText(proxyResp.Status))
	browserConn.Write([]byte(respLine))

	for key, value := range proxyResp.Headers {
		browserConn.Write([]byte(key + ": " + value + "\r\n"))
	}

	browserConn.Write([]byte("\r\n"))
	browserConn.Write([]byte(proxyResp.Body))
}