package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
	"bufio"
)

// Variáveis globais interessantes para o processo
var err string
var myPort string          //porta do meu servidor
var nServers int           //qtde de outros processo
var CliConn map[int]*net.UDPConn // mapa com conexões para os servidores dos outros processos
// dos outros processos
var ServConn *net.UDPConn //conexão do meu servidor (onde recebo
//mensagens dos outros processos)
var logicalClock int
var myID int // Adicione um ID para o processo

func CheckError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}
}

func PrintError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
	}
}

func doServerJob() {
	buf := make([]byte, 1024)
	for {
		n, addr, err := ServConn.ReadFromUDP(buf)
		receivedClock, _ := strconv.Atoi(string(buf[0:n]))
		if receivedClock > logicalClock {
			logicalClock = receivedClock
		}
		logicalClock++
		fmt.Println("Received ", string(buf[0:n]), " from ", addr)
		PrintError(err)
	}
}

func doClientJob(otherProcess int, clock int) {
	msg := strconv.Itoa(clock)
	buf := []byte(msg)
	_, err := CliConn[otherProcess].Write(buf)
	fmt.Printf("Sent Clock: %d to process %d\n", clock, otherProcess+1) // +1 para ajustar o índice para o ID
	PrintError(err)
}

func initConnections() {
    myID, _ = strconv.Atoi(os.Args[1])  // convertendo o ID para inteiro
    myPort = os.Args[myID + 1]  // o porto é agora indexado pelo ID
    nServers = len(os.Args) - 3  // você tem len(os.Args) - 2 servidores, incluindo o próprio servidor
    CliConn = make(map[int]*net.UDPConn)  // não precisamos pré-alocar o mapa
    ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+myPort)
    CheckError(err)
    ServConn, err = net.ListenUDP("udp", ServerAddr)
    CheckError(err)
    j := 0
    for i := 2; i < len(os.Args); i++ {
        if i != myID + 1 {  // não queremos conectar ao nosso próprio servidor
            ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+os.Args[i])
            CheckError(err)

            Conn, err := net.DialUDP("udp", nil, ServerAddr)
            CheckError(err)

            CliConn[j] = Conn  // adicionamos a conexão ao nosso mapa
            j++
        }
    }
}

func readInput(ch chan string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _, _ := reader.ReadLine()
		ch <- string(text)
	}
}

func printConnections(){
	for i, conn := range CliConn {
		if conn != nil {
			fmt.Printf("CliConn[%d] - Local: %s, Remote: %s\n", i, conn.LocalAddr(), conn.RemoteAddr())
		} else {
			fmt.Printf("CliConn[%d] is nil\n", i)
		}
	}
}

func main() {
	myID, _ = strconv.Atoi(os.Args[1])
	fmt.Printf("myID: %d\n", myID)
	initConnections()
	//O fechamento de conexões deve ficar aqui, assim só fecha   //conexão quando a main morrer
	defer ServConn.Close()
	for i := 0; i < nServers; i++ {
		defer CliConn[i].Close()
	}
	ch := make (chan string)
	go readInput(ch)
	logicalClock = 0
	go doServerJob()
	for {
		select {
		case x, valid := <-ch:
			if valid {
				targetID, _ := strconv.Atoi(x)
				printConnections()
				if targetID == myID {
					logicalClock++
					fmt.Printf("Internal operation. New Logical Clock: %d\n", logicalClock)
				} else if targetID > 0 && targetID <= nServers {
					logicalClock++
					go doClientJob(targetID-1, logicalClock) // -1 para ajustar o ID para o índice
				}
			} else {
				fmt.Println("Closed channel !")
			}
		default:
			time.Sleep(time.Second * 1)
		}
		time.Sleep(time.Second * 1)
	}
}
