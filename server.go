package main

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
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


func handleConn(clientConn net.Conn){
	defer clientConn.Close()

	decoder := json.NewDecoder(clientConn)
	encoder := json.NewEncoder(clientConn)

	var req ProxyRequest
	if err := decoder.Decode(&req); err != nil {
		log.Println("リクエスト解析エラー:",err)
		return
	}

	log.Println("[サーバー] クライアントからリクエスト受信:")
	log.Printf("  Method: %s, Host: %s, Path: %s\n", req.Method, req.Host, req.Path)

	// 実際のHTTPリクエストを実行
	resp, err := executeRequest(&req)
	if err != nil {
		log.Println("リクエスト実行エラー:", err)
		encoder.Encode(ProxyResponse{Status: 500, Body: "Error"})
		return
	}

	log.Printf("[サーバー] レスポンス送信: ステータス %d\n", resp.Status)
	// レスポンスをクライアントに送信
	encoder.Encode(resp)
}

func executeRequest(req *ProxyRequest) (*ProxyResponse, error) {
	url := "http://" + req.Host + req.Path

	httpReq, err := http.NewRequest(req.Method, url, nil)
	if err != nil {
		return nil, err
	}

	// ヘッダを設定
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	// レスポンスボディを読む
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	// ヘッダを辞書に変換
	headers := make(map[string]string)
	for key, values := range httpResp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	return &ProxyResponse{
		Status:  httpResp.StatusCode,
		Headers: headers,
		Body:    string(body),
	}, nil
}