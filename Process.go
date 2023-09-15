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
var CliConn []*net.UDPConn //vetor com conexões para os servidores
// dos outros processos
var ServConn *net.UDPConn //conexão do meu servidor (onde recebo
//mensagens dos outros processos)

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
	//Loop infinito mesmo
	for {
		// Ler (uma vez somente) da conexão UDP a mensagem
		n, addr, err := ServConn.ReadFromUDP(buf)
		// Escrever na tela a msg recebida (indicando o
		// endereço de quem enviou)
		fmt.Println("Received ", string(buf[0:n]), " from ", addr)
		PrintError(err)
	}
}

func doClientJob(otherProcess int, i int) {
	// Enviar uma mensagem (com valor i) para o servidor do processo //otherServer.
	msg := strconv.Itoa(i)
	i++
	buf := []byte(msg)
	_, err := CliConn[otherProcess].Write(buf)
	PrintError(err)
}

func initConnections() {
	myPort = os.Args[1]
	nServers = len(os.Args) - 2
	/*Esse 2 tira o nome (no caso Process) e tira a primeira porta (que é a minha). As demais portas são dos outros processos*/
	CliConn = make([]*net.UDPConn, nServers)

	/*Outros códigos para deixar ok a conexão do meu servidor (onde re-cebo msgs). O processo já deve ficar habilitado a receber msgs.*/

	ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+myPort)
	CheckError(err)
	ServConn, err = net.ListenUDP("udp", ServerAddr)
	CheckError(err)

	/*Outros códigos para deixar ok a minha conexão com cada servidor dos outros processos. Colocar tais conexões no vetor CliConn.*/
	for s := 0; s < nServers; s++ {
		ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+os.Args[2+s])
		CheckError(err)

		/*Aqui não foi definido o endereço do cliente.
		  Usando nil, o próprio sistema escolhe.  */
		Conn, err := net.DialUDP("udp", nil, ServerAddr)
		CliConn[s] = Conn
		CheckError(err)
	}
}

func readInput(ch chan string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _, _ := reader.ReadLine()
		ch <- string(text)
	}
}
func main() {

	initConnections()

	//O fechamento de conexões deve ficar aqui, assim só fecha   //conexão quando a main morrer
	defer ServConn.Close()
	for i := 0; i < nServers; i++ {
		defer CliConn[i].Close()
	}

	ch := make (chan string)
	go readInput(ch)

	/*Todo Process fará a mesma coisa: ficar ouvindo mensagens e man-dar infinitos i’s para os outros processos*/
	go doServerJob()
	for {
		select {
		case x, valid := <-ch:
			if valid {
				fmt.Printf("From keyboard: %s \n", x)
				for j := 0; j < nServers; j++{
					go doClientJob(j, 100)
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
