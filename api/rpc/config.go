package rpc

import "time"

var (
	MaxSimultaneousTcpConn = 50 // max simultaneous Tcp connection for one Server

	MaxHttpHeaderBytes     = 10 * 1024 // 10k
	MaxHttpBodyReaderBytes = 10 * 1024 // 10k

	HttpWriteTimeout      = 10 * time.Second
	HttpReadHeaderTimeout = 5 * time.Second

	CertFilePath = "/mnt/c/Users/30640/Desktop/api_server/topia/api/rpc/example/test/server/server-cert.pem" // tmp test path
	KeyFilePath  = "/mnt/c/Users/30640/Desktop/api_server/topia/api/rpc/example/test/server/server-key.pem"  // tmp test path
)
