package main

import (
    "io"
    "net"
    "log"
    "time"
    "sync"
)

var localAddr = "127.0.0.1:1080"
var serverAddr = "127.0.0.1:8080"
var waiter sync.WaitGroup
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
    log.Printf("get header: %v, method %v\r\n", header, methods)

    methodNego := make([]byte, 2)
    methodNego[0] = ver
    methodNego[1] = METHOD_NO_AUTH_REQ
    conn.Write(methodNego)
    log.Printf("start proxy\r\n")
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
    log.Printf("Request ver %v, cmd %v addr type %v\r\n", ver,
        cmd, addr_type)
    serverConn.Write(header)
    if addr_type == ADDR_TYPE_IPV4 {
        dst_addr := make([]byte, 4) 
        conn.Read(dst_addr)
        log.Printf("dst_addr: %v\r\n", dst_addr)
    } else if addr_type == ADDR_TYPE_DOMAINNAME {
        len := make([]byte, 1)
        conn.Read(len)
        domain := make([]byte, len[0])
        conn.Read(domain)
        log.Printf("get domain: %q\r\n", domain)
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
    log.Printf("server accept proxy: %d\r\n", ret)
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

func reader(data_queue chan []byte, src net.Conn) {
    buf := make([]byte, bufSize)
    for {
        cnt, err := src.Read(buf)
        if err == io.EOF || cnt == 0 || err != nil {
            log.Printf("game over: %d\r\n", cnt)
            break
        }

        log.Printf("push %dbytes data into recv_buf\r\n", cnt)
        data_queue <- buf[:cnt]
    }
    waiter.Done()
}

func writer(data_queue chan []byte, conn net.Conn) {
    for {
        buf := <-data_queue
        cnt, err := conn.Write(buf)
        log.Printf("pass %dbytes to server: %v\r\n", cnt, err)
        if err != nil {
            break
        }
    }
    waiter.Done()
}

func transfer(src, dst net.Conn) {
    data_queue := make(chan []byte)
    waiter.Add(2)
    go reader(data_queue, src)
    go writer(data_queue, dst)
}

func forwardData(remote, local net.Conn) {
    go transfer(local, remote)
    go transfer(remote, local)

    waiter.Wait()

    remote.Close()
    local.Close()
}

func handleConnection(serverConn, conn net.Conn) {
    handShake(conn)
    getRequest(serverConn, conn)
    sendReply(conn)
    forwardData(serverConn, conn)
}

func hello2ProxyServer(addr string) (net.Conn, error) {
    log.Printf("start connect to server: %v\r\n", addr)
    server_conn, err := net.Dial("tcp", addr) 
    if err != nil {
        return nil, err
    }
    return server_conn, nil
}

func main() {
    localProxy, err := net.Listen("tcp", localAddr)
    if err != nil {
         log.Fatal("Create local proxy server failed: %v\r\n", err)
    }

    for {
        conn, err := localProxy.Accept()
        if err != nil {
             log.Printf("Failed to accept new client.\r\n")
             break
        }

        var serverConn net.Conn
        for {
            serverConn, err = hello2ProxyServer(serverAddr)
            if err == nil {
                log.Printf("connect to proxy server %s: ok\r\n", serverAddr)
                break
            } else {
                log.Printf("failed to connect proxy server: %v\r\n", err)
                time.Sleep(1)
            }
        }

        handleConnection(serverConn, conn)
    }
}
