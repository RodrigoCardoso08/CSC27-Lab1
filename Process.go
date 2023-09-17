package main
import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
	"bufio"
	"sync"
	"encoding/json"
	"container/list"
)
var err string
var myPort string
var nServers int
var CliConn map[int]*net.UDPConn
var ServConn *net.UDPConn
var logicalClock int
var wantedClock int
var myID int
var clockMutex sync.Mutex
const (
	RELEASED = iota
	WANTED
	HELD
)
var state int
type Message struct {
	ID    int `json:"id"`
	Clock int `json:"clock"`
	ReceivedClock int `json:"clock"`
	Type  string `json:"type"`
}
var requestQueue *list.List
var repliesReceived []bool
type ExtendedMessage struct {
	ID    int    `json:"id"`
	Clock int    `json:"clock"`
	Type  string `json:"type"`
	Text  string `json:"text"`
}

func requestAccessToCS() {
    if state == HELD || state == WANTED {
        fmt.Println("x ignorado")
        return
    }
    state = WANTED
	for i := range repliesReceived {
		if i != myID && i != 0 { repliesReceived[i] = false }
	}
	updateClock(0)
	msg := Message{ ID: myID, Clock: logicalClock, Type: "request"}
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}
	buf := []byte(msgJSON)
	for i, conn := range CliConn {
		if conn != nil {
			_, err := conn.Write(buf)
			if err == nil {
				fmt.Printf("Sent message: %s to process %d\n", string(msgJSON), i)
			} else {
				PrintError(err)
			}
		} else {
			fmt.Printf("CliConn[%d] is nil\n", i)
		}
	}
	for {
		allReceived := true
		for _, received := range repliesReceived {
			if !received {
				allReceived = false
				break
			}
		}
		if allReceived {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}
	state = HELD
	fmt.Println("Entrei na CS\n")
	sharedResourceAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:10001")
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}
	conn, err := net.DialUDP("udp", nil, sharedResourceAddr)
	if err != nil {
		fmt.Println("Error dialing:", err)
		return
	}
	defer conn.Close()
	sharedResourceMsg := ExtendedMessage{
		ID:    myID,
		Clock: logicalClock,
		Type:  "sharedResource",
		Text:  "Oi...",
	}
	extendedMsgJSON, err := json.Marshal(sharedResourceMsg)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}
	_, err = conn.Write(extendedMsgJSON)
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}
	time.Sleep(time.Second * 5)
	fmt.Println("Sai da CS\n")
	state = RELEASED
	for e := requestQueue.Front(); e != nil; e = e.Next() {
		request := e.Value.(Message)
		conn = CliConn[request.ID]
		msg := Message{
			ID:    myID,
			Clock: logicalClock,
			ReceivedClock: request.ReceivedClock,
			Type:  "reply",
		}
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("Error marshalling JSON:", err)
			return
		}
		buf := []byte(msgJSON)
		if conn != nil {
			_, err := conn.Write(buf)
			if err != nil {
				fmt.Println("Error marshalling JSON:", err)
				return
			}
		}
	}
	requestQueue.Init()
}

func updateClock(receivedClock int) {
    clockMutex.Lock()
    logicalClock = 1 + max(logicalClock, receivedClock)
    clockMutex.Unlock()
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

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
		n, _, err := ServConn.ReadFromUDP(buf)
		if err != nil {
			PrintError(err)
			continue
		}
		var receivedMsg Message
		err = json.Unmarshal(buf[:n], &receivedMsg)
		if err != nil {
			fmt.Println("Error unmarshalling JSON:", err)
			continue
		}
		fmt.Printf("Received message from: ID=%d, Clock=%d, Type=%s\n", receivedMsg.ID, receivedMsg.Clock, receivedMsg.Type)
		if receivedMsg.Type == "request" {
			updateClock(receivedMsg.Clock)
			if state == HELD || (state == WANTED && (wantedClock < receivedMsg.Clock || (wantedClock == receivedMsg.Clock && myID < receivedMsg.ID))) {
				requestQueue.PushBack(receivedMsg)
			} else {
				var conn = CliConn[receivedMsg.ID]
				msg := Message{
					ID:    myID,
					Clock: logicalClock,
					ReceivedClock: receivedMsg.ReceivedClock,
					Type:  "reply",
				}
				msgJSON, err := json.Marshal(msg)
				if err != nil {
					fmt.Println("Error marshalling JSON:", err)
					return
				}
				buf := []byte(msgJSON)
				if conn != nil {
					_, err := conn.Write(buf)
					if err != nil {
						fmt.Println("Error writing to connection:", err)
						return
					}
				}
			}
		} else if receivedMsg.Type == "reply" {
			repliesReceived[receivedMsg.ID] = true
			wantedClock = receivedMsg.ReceivedClock
			updateClock(receivedMsg.Clock)
		}
	}
}

func doClientJob(targetID int, clock int) {
    msg := strconv.Itoa(clock)
    buf := []byte(msg)
    if conn, ok := CliConn[targetID]; ok {
        _, err := conn.Write(buf)
        fmt.Printf("Sent Clock: %d to process %d\n", clock, targetID)
        PrintError(err)
    } else {
        fmt.Printf("No connection found for process ID %d\n", targetID)
    }
}

func initConnections() {
    myID, _ = strconv.Atoi(os.Args[1])
    myPort = os.Args[myID + 1]
    nServers = len(os.Args) - 3
    CliConn = make(map[int]*net.UDPConn)
    ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+myPort)
    CheckError(err)
    ServConn, err = net.ListenUDP("udp", ServerAddr)
    CheckError(err)
    for i := 0; i <= nServers; i++ {
        if i != myID - 1 {
            ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1"+os.Args[i+2])
            CheckError(err)
            Conn, err := net.DialUDP("udp", nil, ServerAddr)
            CheckError(err)
            CliConn[i+1] = Conn
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
	initConnections()
	state = RELEASED
	requestQueue = list.New()
	defer ServConn.Close()
	for _, conn := range CliConn {
		defer conn.Close()
	}
	ch := make(chan string)
	go readInput(ch)
	logicalClock = 0
	repliesReceived = make([]bool, nServers+2)
	repliesReceived[0] = true
	repliesReceived[myID] = true
	go doServerJob()
	for {
		select {
		case x, valid := <-ch:
			if valid {
				if x == "x" {
					requestAccessToCS()
				} else {
					targetID, _ := strconv.Atoi(x)
					printConnections()
					if targetID == myID {
						clockMutex.Lock()
						logicalClock++
						clockMutex.Unlock()
						fmt.Printf("Internal operation. New Logical Clock: %d\n", logicalClock)
					} else if _, ok := CliConn[targetID]; ok {
						clockMutex.Lock()
						logicalClock++
						clockMutex.Unlock()
						go doClientJob(targetID, logicalClock)
					} else {
						fmt.Printf("Invalid target ID: %d\n", targetID)
					}
				}
			} else {
				fmt.Println("Closed channel!")
			}
		default:
			time.Sleep(time.Second * 1)
		}
		time.Sleep(time.Second * 1)
	}
}
