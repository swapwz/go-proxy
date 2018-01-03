package main

import (
    "fmt"
    "net"
    "io"
    "encoding/binary"
)

var listenAddr = "0.0.0.0:61987"

func acceptClient(conn net.Conn) {
    var tunnel net.Conn
    header := make([]byte, 4)
    conn.Read(header)
    fmt.Printf("get new header %v\r\n", header)
    if header[3] == 0x03 {
        length := make([]byte, 1)
        conn.Read(length)
        fmt.Printf("Get length %v\r\n", length[0])
        domain := make([]byte, length[0])
        conn.Read(domain)
        fmt.Printf("Get domain %q\r\n", domain)
        port := make([]byte, 2)
        conn.Read(port)
        port_num := binary.BigEndian.Uint16(port)
        fmt.Printf("Get port %d\r\n", port_num)
        targetAddr := fmt.Sprintf("%s:%d", string(domain[:]), port_num)
        fmt.Printf("get target addr: %v\r\n", targetAddr)
        target, err := net.Dial("tcp", targetAddr)
        if err != nil {
           fmt.Printf("err dial to %s: %v\r\n", string(domain), err)
           return
        }
        fmt.Printf("connect to target ok: %v\r\n", target)
        tunnel = target
        ret := make([]byte, 1)
        ret[0] = 'c'
        conn.Write(ret)
    }

    fmt.Printf("forwarding data for client\r\n")

    // start forwarding data
    for {
        lens := make([]byte, 4)
        cnt, err := conn.Read(lens)
        length := binary.BigEndian.Uint32(lens)
        fmt.Printf("get length %d\r\n", length)
        buf := make([]byte, length)
        cnt, err = conn.Read(buf)
        if err == nil {
            fmt.Printf("target %v, receive data %dbytes\r\n", tunnel, cnt)
            tunnel.Write(buf)
        } else if err == io.EOF {
            fmt.Printf("receive data end\r\n")
            break
        }
    }

    max_buf := make([]byte, 1024)
    for {
        fmt.Printf("communicate with target\r\n")
        cnt, err := tunnel.Read(max_buf)
        fmt.Printf("get %dbytes data back\r\n", cnt)
        if err == nil {
            conn.Write(max_buf[:cnt])
        } else if err == io.EOF {
            fmt.Printf("get all data back\r\n")
            break
        }
    }

    fmt.Printf("End forwarding\r\n")
}

func main() {
    server, err := net.Listen("tcp", listenAddr) 
    if err != nil {
        fmt.Printf("failed to start proxy server: %v\r\n", err)
        return
    }

    for {
        conn, err := server.Accept() 
        if err != nil {
            fmt.Print("failed to accept: %v\r\n", err)            
        }
        acceptClient(conn)
    }
}
