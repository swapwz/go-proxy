package main

import (
    "fmt"
    "net"
    "io"
    "encoding/binary"
)

var listenAddr = "0.0.0.0:61987"

func reader(data_queue chan []byte, done chan bool, conn net.Conn) {
    buf := make([]byte, 4096)
    for {
        cnt, err := conn.Read(buf)
        if err == io.EOF || cnt == 0 || err != nil {
            done <- true
            fmt.Printf("game over\r\n")
            break
        }
    
        fmt.Printf("push %dbytes data into recv_buf\r\n", cnt)
        data_queue <- buf[:cnt]       
    }
}

func writer(data_queue chan []byte, conn net.Conn) {
    for {
        buf := <- data_queue
        cnt, err := conn.Write(buf)
        fmt.Printf("pass %dbytes to server: %v\r\n", cnt, err)
        if err != nil {
            break     
        }
    }    
}

func transfer(src, dst net.Conn) {
    data_queue := make(chan []byte)
    done := make(chan bool, 1)

    go reader(data_queue, done, src)
    go writer(data_queue, dst)
    <- done 
    dst.Close()
    fmt.Printf("End forwarding\r\n")
}

func forwardData(local, remote net.Conn) {
    fmt.Printf("forwarding data for client\r\n")
    go transfer(local, remote)
    go transfer(remote, local)
}

func acceptClient(conn net.Conn) {
    header := make([]byte, 4)
    conn.Read(header)
    fmt.Printf("get new header %v\r\n", header)
    if header[3] != 0x03 {
        fmt.Printf("not support\r\n")
        return
    }

    length := make([]byte, 1)
    conn.Read(length)
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

    // send ok back to client
    fmt.Printf("connect to target ok: %v\r\n", target)
    ret := make([]byte, 1)
    ret[0] = 'c'
    conn.Write(ret)

    forwardData(conn, target)
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
