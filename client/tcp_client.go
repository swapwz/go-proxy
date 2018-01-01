package main

import (
    "fmt"
    "io"
    "net"
    "log"
    "encoding/binary"
)

var localAddr = "127.0.0.1:1080"
var serverAddr = "www.namespace.cc:61987"
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
    ADDR_TYPE_DOMAINNAME = 0x03
    ADDR_TYPE_IPV6 = 0x04
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
func getRequest(serverConn, conn net.Conn) error {
    header := make([]byte, 4) 
    conn.Read(header)  
    ver := header[0]
    cmd := header[1]
    addr_type := header[3]
    fmt.Printf("Request ver %v, cmd %v addr type %v\r\n", ver,
        cmd, addr_type)
    serverConn.Write(header)
    if addr_type == ADDR_TYPE_IPV4 {
        dst_addr := make([]byte, 4) 
        conn.Read(dst_addr)
        fmt.Printf("dst_addr: %v\r\n", dst_addr)
    } else if addr_type == ADDR_TYPE_DOMAINNAME {
        len := make([]byte, 1)
        conn.Read(len)
        domain := make([]byte, len[0])
        conn.Read(domain)
        fmt.Printf("get domain: %q\r\n", domain)
        serverConn.Write(len)
        serverConn.Write(domain)
    }
    port := make([]byte, 2)
    _, err := conn.Read(port)
    if err != nil {
        return err    
    }

    serverConn.Write(port)
    ret := make([]byte, 1)
    serverConn.Read(ret)
    fmt.Printf("server accept proxy: %d\r\n", ret)
    return nil
}

func sendReply(conn net.Conn) {
    reply := make([]byte, 12)
    reply[0] = 0x5
    reply[1] = 0x0
    reply[2] = 0x0
    reply[3] = 0x1
    reply[4] = 127
    reply[5] = 0
    reply[6] = 0
    reply[7] = 1
    reply[9] = 0x80
    reply[10] = 0x0
    conn.Write(reply)
}

func forwardData(serverConn, conn net.Conn) {
    buf := make([]byte, 4096)
    end := make([]byte, 1)
    end[0] = 0
    for  {
        cnt, err := conn.Read(buf[4:])    
        fmt.Printf("forwarding %dbytes data\r\n", cnt)
        length := uint32(cnt)
        // convert length to first 4 bytes
        binary.BigEndian.PutUint32(buf[:4], length)
        if cnt != 0 && err == nil {
            cnt, err = serverConn.Write(buf[:4+cnt])
            fmt.Printf("pass %dbytes to server: %v\r\n", cnt, err)
        } else if cnt == 0 || err == io.EOF {
            fmt.Printf("forwarding over, waiting for server response: %v\r\n", err)
            for {
                cnt, err := serverConn.Read(buf)
                fmt.Printf("get %dbytes data from server\r\n", cnt)
                if err == nil {
                    conn.Write(buf[:cnt])
                } else if err == io.EOF {
                    fmt.Printf("get data over\r\n")   
                    conn.Write(end)
                    conn.Close() 
                    return
                }
            }
        } 
    }
}

func handleConnection(serverConn net.Conn, conn net.Conn) {
    handShake(conn)
    getRequest(serverConn, conn)
    sendReply(conn)
    forwardData(serverConn, conn)
}

func hello2ProxyServer(addr string) (net.Conn, error) {
    fmt.Printf("start connect to server: %v\r\n", addr)
    server_conn, err := net.Dial("tcp", addr) 
    if err != nil {
        return nil, err
    }
    return server_conn, nil
}

func main() {
    serverConn, err := hello2ProxyServer(serverAddr)
    if err != nil {
        fmt.Printf("failed to connect proxy server: %v\r\n", err)     
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
             break
        }
        handleConnection(serverConn, conn)
    }
}
