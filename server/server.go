package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
)

type heads struct {
	udpAddr *net.UDPAddr
	path    string
	seqN    uint32
	ackN    uint32
	connID  uint16
	notUsed uint16
	ack     uint
	syn     uint
	fin     uint
}

const (
	maxDatagSize = 524
	payload      = 500
	maxSeq       = 102400
	maxRec       = 102400
	minCWND      = 512
	limSSTHRESH  = 10000
)

var (
	list    = []heads{}
	counter = 1
)

func createHead(src *net.UDPAddr) heads {
	var head heads
	head.udpAddr = src
	head.seqN = 4321
	head.ackN = 0
	head.connID = 0
	head.notUsed = 0
	head.ack = 0
	head.syn = 0
	head.fin = 0

	return head
}

func translateHead(msg []byte) heads {
	var head heads
	head.seqN = binary.LittleEndian.Uint32(msg[0:4])
	head.ackN = binary.LittleEndian.Uint32(msg[4:8])
	head.connID = binary.LittleEndian.Uint16(msg[8:12])
	head.ack = uint(binary.LittleEndian.Uint16(msg[12:16]))
	head.syn = uint(binary.LittleEndian.Uint16(msg[16:20]))
	head.fin = uint(binary.LittleEndian.Uint16(msg[20:24]))

	return head
}

func createMsg(head heads) *bytes.Buffer {
	msg := make([]byte, 4)
	buf := bytes.NewBuffer(nil)
	binary.LittleEndian.PutUint32(msg, head.seqN)
	buf.Write(msg)
	binary.LittleEndian.PutUint32(msg, head.ackN)
	buf.Write(msg)
	binary.LittleEndian.PutUint16(msg, head.connID)
	buf.Write(msg)
	binary.LittleEndian.PutUint16(msg, uint16(head.ack))
	buf.Write(msg)
	binary.LittleEndian.PutUint16(msg, uint16(head.syn))
	buf.Write(msg)
	binary.LittleEndian.PutUint16(msg, uint16(head.fin))
	buf.Write(msg)

	return buf
}

func saveFile(head heads, bFile []byte) {
	file, err := os.Open(head.path)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()
	// file.WriteAt(bFile, int64(head.ackN))
	// file.WriteString(string(bFile))
	// file.Write(bFile)
	buffer := bufio.NewReader(file)
	buf := make([]byte, len(bFile))
	buffer.Read(buf)
	w := bufio.NewWriter(file)
	w.Write(buf)

	return
}

func updateList(head heads) []heads {
	var l []heads
	for _, v := range list {
		if v.connID == head.connID {
			continue
		} else {
			l = append(l, v)
		}
	}
	return l
}

func closeConnection(udpConn *net.UDPConn, head heads) {
	reply := make([]byte, maxDatagSize)
	head.ack = 1
	buf := createMsg(head)
	buf.Read(reply)
	buf.Reset()
	udpConn.WriteToUDP(reply, head.udpAddr)
	head.ack = 0
	head.fin = 0
	head.ackN = 0

	for {
		buf = createMsg(head)
		buf.Read(reply)
		buf.Reset()
		udpConn.WriteToUDP(reply, head.udpAddr)
		udpConn.ReadFromUDP(reply)
		head := translateHead(reply[0:24])
		if head.ack == 1 {
			udpConn.Close()
			list = updateList(head)
			// return
		}
	}
}

func handleClient(msg []byte, head heads) {
	conn, err := net.DialUDP("udp", nil, head.udpAddr)
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}
	defer conn.Close()

	conn.WriteToUDP(msg, head.udpAddr)

	for {
		reply := make([]byte, maxDatagSize)
		msg := make([]byte, maxDatagSize)
		fmt.Println("Waiting package from client", head.udpAddr)
		conn.ReadFromUDP(msg)
		headAux := translateHead(msg[0:24])

		if headAux.connID == head.connID {
			fmt.Println("Recived package from client", head.udpAddr)
			if headAux.ack == 0 {
				headAux.ack = 1
			}
			file := msg[24:]

			if headAux.fin == 1 {
				head.fin = 1
				saveFile(head, file)
				fmt.Println("Closing connection with client", head.udpAddr)
				closeConnection(conn, head)
			} else {
				if headAux.ackN == head.seqN {
					saveFile(headAux, file)
					head.ackN = headAux.seqN + uint32(payload)
					head.ack = 1
					buf := createMsg(head)
					buf.Read(reply)
					buf.Reset()
					fmt.Println("Sending ACK", head.ackN, "to client", head.udpAddr)
					conn.WriteToUDP(reply, head.udpAddr)
				} else {
					fmt.Println("Wrong package from client", head.udpAddr)
					//Não sei se é isso que eu devo fazer
					buf := createMsg(head)
					buf.Read(reply)
					buf.Reset()
					conn.WriteToUDP(reply, head.udpAddr)
				}
			}
		}
	}
}

func handshake(udpAddr *net.UDPAddr, msg []byte) {
	// defer handshake.Close()

	service := "224.0.0.1:0"
	udp, err := net.ResolveUDPAddr("udp", service)
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}

	head := createHead(udpAddr)
	reply := make([]byte, maxDatagSize)
	// fmt.Println("Waiting package with SYN from client ", src)
	// listener.ReadFrom(reply)
	fmt.Println("Recived package with SYN from client ", udpAddr)
	head = translateHead(msg)

	if head.syn == uint(1) {
		conn, err := net.ListenUDP("udp", udp)
		if err != nil {
			log.Fatal(err)
			os.Exit(-1)
		}
		defer conn.Close()

		head.ack = 1
		head.connID = uint16(counter)
		counter++
		// head.seqN++
		buf := createMsg(head)
		send := make([]byte, buf.Len())
		buf.Read(send)
		// buf.Reset()
		fmt.Println("Sending package with ACK and SYN to client", udpAddr)
		conn.WriteToUDP(send, udpAddr)

		fmt.Println("Waiting package with ACK from client", udpAddr)
		conn.ReadFromUDP(reply)
		head = translateHead(reply)

		if head.ack == 1 {
			fmt.Println("Recived package with ACK from client", udpAddr)
			head.udpAddr = udpAddr
			fmt.Println("Handshake with client", udpAddr, "was succeful")
			head.path = fmt.Sprint(head.connID)
			head.path += ".file"
			os.Create(head.path)
			fmt.Println("Created ", head.path)
			file := reply[24:]
			ioutil.WriteFile(head.path, file, 0666)
			list = append(list, head)
			buf = createMsg(head)
			buf.Read(send[0:])
			fmt.Println("Sending package with ACK", head.ackN, "to client", head.udpAddr)
			// conn.WriteToUDP(send, head.udpAddr)
			conn.Close()
			handleClient(send, head)
			// return
		} else {
			fmt.Println("Handshake with client", udpAddr, "failed")
		}
	} else {
		fmt.Println("Handshake with client", udpAddr, "failed")
		// return
	}
}

func serverMulticast(port string) {
	service := "224.0.0.1:"
	service += port
	addr, err := net.ResolveUDPAddr("udp", service)
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}

	listener, err := net.ListenMulticastUDP("udp", nil, addr)
	// listener, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}
	listener.SetReadBuffer(maxDatagSize)
	defer listener.Close()
	for {
		msg := make([]byte, maxDatagSize)
		_, src, _ := listener.ReadFromUDP(msg)
		fmt.Println("Iniciating handshake with client", src)
		go handshake(src, msg)
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Error invalid command")
		os.Exit(-1)
	}

	fileDir := os.Args[2]
	err := os.Chdir(fileDir)
	var input string
	if err != nil {
		fmt.Println("Directory invalid, do you wish to create a directory? y/n")
		_, err := fmt.Scanln(input)

		if input == "n" || err != nil {
			fmt.Println("Closing application")
			os.Exit(1)
		} else if input == "y" {
			os.Mkdir(fileDir, os.ModeDir)
			fmt.Println("Created directoy ", fileDir)
			os.Chdir(fileDir)
		}
	}

	fmt.Println("Application started")
	port := os.Args[1]
	// go handleClient()
	serverMulticast(port)
}
