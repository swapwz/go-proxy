package main

import (
    "fmt"
    "net"
    "io"
    "encoding/binary"
)

var listenAddr = "0.0.0.0:8080"

func reader(data_queue chan []byte, done chan bool, conn net.Conn) {
    buf := make([]byte, 4096)
    for {
        cnt, err := conn.Read(buf)
        if err == io.EOF || cnt == 0 || err != nil {
            var eof [1]byte
            eof[0] = 0
            done <- true
            fmt.Printf("game over\r\n")
            data_queue <- eof[:]
            break
        }

        fmt.Printf("push %d bytes data into recv_buf\r\n", cnt)
        fmt.Printf("Read from client: %v\r\n", buf[:cnt])
        data_queue <- buf[:cnt]
    }
}

func writer(data_queue chan []byte, conn net.Conn) {
    for {
        buf := <-data_queue
        if len(buf) == 1 && buf[0] == 0 {
            fmt.Printf("Peer over\r\n")
            break
        }
        cnt, err := conn.Write(buf)
        fmt.Printf("pass %d bytes to server\r\n", cnt)
        if err != nil {
            fmt.Printf("Peer close\r\n")
            break
        }
    }
}

func transfer(src, dst net.Conn, done chan bool) {
    data_queue := make(chan []byte)

    go reader(data_queue, done, src)
    go writer(data_queue, dst)
}

func forwardData(local, remote net.Conn) {
    fmt.Printf("Begin forwarding\r\n")
    read_done := make(chan bool, 1)
    write_done := make(chan bool, 1)
    go transfer(local, remote, read_done)
    go transfer(remote, local, write_done)
    <-read_done
    <-write_done
    local.Close()
    remote.Close()
    fmt.Printf("End forwarding\r\n")
}

func runProxyV4(conn net.Conn) {
    command := make([]byte, 1)

    fmt.Printf("Run proxy v4 for this connection.\r\n")

    conn.Read(command)
    fmt.Printf("Get command %v.\r\n", command)
    /* CONNECT Request */
    if command[0] == 0x01 {
        var dstPort [2]byte
        var dstIP [4]byte

        conn.Read(dstPort[:])
        conn.Read(dstIP[:])

        port := binary.BigEndian.Uint16(dstPort[:])
        fmt.Printf("Get dstIP %v and port %v.\r\n", dstIP, port)

        targetAddr := fmt.Sprintf("%d.%d.%d.%d:%d", dstIP[0], dstIP[1],
            dstIP[2], dstIP[3], port)
        fmt.Printf("Dial to %v\r\n", targetAddr);

        var response [8]byte
        response[0] = 0

        target, err := net.Dial("tcp", targetAddr)
        if err != nil {
            response[1] = 91
            fmt.Printf("Failed to connect %q\r\n", targetAddr)
            conn.Write(response[:])
            conn.Close()
            return
        } else {
            response[1] = 90
            fmt.Printf("Connect target OK.\r\n")
            var octect [1]byte
            octect[0] = 1
            for octect[0] != 0 {
                conn.Read(octect[:])
                fmt.Printf("Read: %q\r\n", octect)
            }
            conn.Write(response[:])
        }
        //forwardData(conn, target)
        go io.Copy(conn, target)
        go io.Copy(target, conn)
    } else {
        fmt.Printf("Not supported request:%v.\r\n", command[0])
    }

    return
}

func runProxyV5(conn net.Conn) {
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

    //forwardData(conn, target)
    return
}

func notImplemented(conn net.Conn) {
    fmt.Printf("Not supported version number\r\n")
    conn.Close()
}

func acceptClient(conn net.Conn) {
    version_number := make([]byte, 1)
    conn.Read(version_number)

    fmt.Printf("Get proxy version: %v\r\n", version_number);
    if version_number[0] == 4 {
        runProxyV4(conn)
    } else if version_number[0] == 5 {
        runProxyV5(conn)
    } else {
        notImplemented(conn)
    }

    return
}

func main() {
    server, err := net.Listen("tcp", listenAddr)
    if err != nil {
        fmt.Printf("failed to start proxy server: %v\r\n", err)
        return
    }

    for {
        conn, err := server.Accept()
        fmt.Printf("Get a new connection\r\n")
        if err != nil {
            fmt.Print("failed to accept: %v\r\n", err)
        } else {
            acceptClient(conn)
            fmt.Printf("Process connection over\r\n")
        }
    }
}
