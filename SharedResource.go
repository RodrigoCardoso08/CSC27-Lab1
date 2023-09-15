package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)
type ExtendedMessage struct {
	ID    int    `json:"id"`
	Clock int    `json:"clock"`
	Type  string `json:"type"`
	Text  string `json:"text"`
}

func CheckError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}
}

func main() {
	Address, err := net.ResolveUDPAddr("udp", ":10001")
	CheckError(err)
	Connection, err := net.ListenUDP("udp", Address)
	CheckError(err)
	defer Connection.Close()
	buf := make([]byte, 1024)
	for {
		// Loop infinito para receber mensagem e escrever todo
		// conteúdo (processo que enviou, relógio recebido e texto)
		// na tela
		n, addr, err := Connection.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		var receivedMsg ExtendedMessage
		err = json.Unmarshal(buf[:n], &receivedMsg)
		if err != nil {
			fmt.Println("Error unmarshalling JSON:", err)
			continue
		}

		fmt.Printf("Received message from %s: ID=%d, Clock=%d, Type=%s\n", addr, receivedMsg.ID, receivedMsg.Clock, receivedMsg.Type)
	}
}
