package main

import (
    "fmt"
    "log"
    "net"
    "encoding/binary"
)

var listenAddr = "0.0.0.0:8080"

func init() {
    log.SetPrefix("[GOPROXY SERVER]")
}

func transfer(src, dst net.Conn) {
    var buf [8192]byte

    for {
        cnt, err := src.Read(buf[:])
        if cnt > 0 {
            cnt, err = dst.Write(buf[:cnt])
            if err != nil {
                fmt.Printf("Write error\r\n")
                dst.Close()
                break
            }
        }
        if cnt == 0 {
            fmt.Printf("Read over\r\n")
            break
        }

        if err != nil {
           fmt.Printf("Read error")
           src.Close()
           break
        }
    }
}


func runProxyV4(conn net.Conn) {
    command := make([]byte, 1)

    log.Printf("Run proxy v4 for this connection.\r\n")

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
            go transfer(conn, target)
            go transfer(target, conn)
        }
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

    go transfer(conn, target)
    go transfer(target, conn)

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
        if err != nil {
            fmt.Print("failed to accept: %v\r\n", err)
        } else {
            fmt.Printf("Get a new connection\r\n")
            go acceptClient(conn)
        }
    }
}
