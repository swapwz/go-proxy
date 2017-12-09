package main

import (
    "fmt"
    "net"
    "log"
)

var localAddr = "127.0.0.1:1080"
var serverAddr = "www.namespace.cc:1987"
const (
   bufSize = 4096
)

/*
 The client connects to the server, and sends a version identifier/method selection message
 +----+----------+----------+
 |VER | NMETHODS | METHODS  |
 +----+----------+----------+
 |  1 |     1    | 1 to 255 |
 +----+----------+----------+
*/
const ( 
    METHOD_NO_AUTH_REQ = 0x0
    METHOD_GSSAPI = 0x01
    METHOD_USERNAME_PASS = 0x02

    REQ_CMD_CONNECT = 0x01
    REQ_CMD_BIND = 0x02
    REQ_UDP_ASSOCIATE = 0x03
    RSV = 0x0
    ADDR_TYPE_IPV4 = 0x01
    ADDR_TYPE_DOMAINNAME = 0x02
    ADDR_TYPE_IPV6 = 0x03
)

func handShake(conn net.Conn) {
    header := make([]byte, 2)
    conn.Read(header)
    ver := header[0]
    nmethods := header[1]
    methods := make([]byte, nmethods)
    conn.Read(methods)
    fmt.Printf("get header: %v, method %v\r\n", header, methods)

    methodNego := make([]byte, 2)
    methodNego[0] = ver
    methodNego[1] = METHOD_NO_AUTH_REQ
    conn.Write(methodNego)
    fmt.Printf("start proxy\r\n")
}

/* 
request format
+----+-----+-------+------+----------+----------+
|VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
+----+-----+-------+------+----------+----------+
| 1  |  1  | X'00' |  1   | Variable |    2     |
+----+-----+-------+------+----------+----------+
*/
func getRequest(conn net.Conn) {
    header := make([]byte, 4) 
    conn.Read(header)  
    ver := header[0]
    cmd := header[1]
    addr_type := header[3]
    fmt.Printf("Request ver %v, cmd %v add type %v\r\n", ver,
        cmd, addr_type)
    if addr_type == 0x01 {
        dst_addr := make([]byte, 4) 
        conn.Read(dst_addr)
        fmt.Printf("dst_addr: %v\r\n", dst_addr)
    } else if addr_type == 0x03 {
        len := make([]byte, 1)
        conn.Read(len)
        domain := make([]byte, len[0])
        conn.Read(domain)
        fmt.Printf("domain: %v\r\n", domain)
    }
    port := make([]byte, 2)
    conn.Read(port)
    fmt.Printf("port %v\r\n", port)
}

func handleConnection(conn net.Conn) {
    handShake(conn)
    getRequest(conn)
}

func hello2ProxyServer(addr string) (net.Conn, error) {
    server_conn, err := net.Dail("tcp", addr) 
    if err != nil {
        return nil, err
    }
    return server_conn, nil
}

func main() {
    server_conn,err := hello2ProxyServer(serverAddr)
    if err != nil {
        fmt.Printf("failed to connect proxy server: %v\r\n, err)     
        return
    }

    fmt.Printf("connect to proxy server %s: ok\r\n", serverAddr)

    localProxy, err := net.Listen("tcp", localAddr)
    if err != nil {
         log.Fatal("Create local proxy server failed: %v\r\n", err)     
    }

    for {
        conn, err := localProxy.Accept()     
        if err != nil {
             fmt.Printf("Failed to accept new client.\r\n")     
             return
        }
        go handleConnection(conn)
    }
}
