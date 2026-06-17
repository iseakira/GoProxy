package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"time"
)

func main() {

	// certs フォルダと証明書ファイルの確認
	if _, err := os.Stat("certs/server.crt"); err == nil {
		if _, err := os.Stat("certs/server.key"); err == nil {
			println("証明書は既に存在します")
			return
		}
	}

	// certsディレクトリがなければ作成
	os.MkdirAll("certs", 0755)

	//秘密鍵の生成
	private, err := ecdsa.GenerateKey(elliptic.P256(),rand.Reader)
	if err != nil {
		panic(err)
	}

	//証明書のテンプレート
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:pkix.Name{
			Organization: []string{"MyVPN"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid:true,
	}

	//証明書の生成
	certDER,err := x509.CreateCertificate(rand.Reader, &template, &template, &private.PublicKey, private)

	if err != nil {
		panic(err)
	}

	//certs/server.crtに保存
	certFile,err := os.Create("certs/server.crt")
	if err != nil {
		panic(err)
	}

	//pem形式とはデジタル証明書や暗号かぎなどのデータをBase64でテキスト化して
	//人間やテキストエディタで読み取れるようなファイル形式
	pem.Encode(certFile, &pem.Block{Type:"CERTIFICATE", Bytes: certDER})
	certFile.Close()

	privateDER,err := x509.MarshalECPrivateKey(private)
	if err != nil {
		panic(err)
	}

	//秘密鍵をDERに変換し
	//DERというバイナリーファイル形式。
	//①人間が読める必要はない②暗号化して保存するのでバイナルが楽
	keyFile,err := os.Create("certs/server.key")
	if err != nil {
		panic(err)
	}
	//pem形式で保存
	pem.Encode(keyFile, &pem.Block{Type:"EC PRIVATE KEY",Bytes:privateDER})
	keyFile.Close()

	println("証明書を作成したよ！")

}