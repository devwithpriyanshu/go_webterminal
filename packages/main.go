package main

import (
    "fmt"
    "log"
    "net/http"
    "os/exec"

    "github.com/creack/pty"
    "github.com/gorilla/websocket"
)

type Message struct {
    Type string `json:"type"`
    Data string `json:"data"`
}

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

func main() {
    port := 5555

    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            log.Println("Upgrade error:", err)
            return
        }
        defer conn.Close()

        cmd := exec.Command("zsh")
        ptyProcess, err := pty.Start(cmd)
        if err != nil {
            log.Println("Failed to start pty:", err)
            return
        }
        defer ptyProcess.Close()

        /*
            goroutine, runs concurrently
            this one has infinite loop, so it keeps on reading data
            from psuedo-terminal process and sends it to websocket connection
        */
        go func() {
            buf := make([]byte, 1024) // temporarily stores data read from the PTY process
            for {
                n, err := ptyProcess.Read(buf)
                if err != nil {
                    log.Println("PTY read error:", err)
                    break
                }

                message := Message{Type: "data", Data: string(buf[:n])}
                if err := conn.WriteJSON(message); err != nil {
                    log.Println("WebSocket write error:", err)
                    break
                }
            }
        }()

        /*
            infinite loop
            reads from websocket connection and
            writes to psuedo-terminal
        */
        for {
            var msg Message
            if err := conn.ReadJSON(&msg); err != nil {
                log.Println("WebSocket read error:", err)
                break
            }

            if msg.Type == "command" {
                _, err := ptyProcess.Write([]byte(msg.Data))
                if err != nil {
                    log.Println("PTY write error:", err)
                }
            }
        }
    })

    fmt.Printf("Listening on port %d...\n", port)
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}