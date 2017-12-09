package main

import (
    "fmt"
)

var listenAddr := "0.0.0.0:1987"

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
        go acceptClient(server)
    }
}
