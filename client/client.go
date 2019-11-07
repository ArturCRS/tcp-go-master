package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

const (
	maxDatagSize = 524
	payload      = 500
	maxSeq       = 102400
	maxRec       = 102400
	minCWND      = 512
	limSSTHRESH  = 10000
)

var (
	cwnd     = minCWND
	ssthresh = limSSTHRESH
)

type headc struct {
	seqN    uint32
	ackN    uint32
	connID  uint16
	notUsed uint16
	ack     uint
	syn     uint
	fin     uint
}

func createHead() headc {
	var head headc
	head.seqN = 12345
	head.ackN = 0
	head.connID = 0
	head.notUsed = 0
	head.ack = 0
	head.syn = 0
	head.fin = 0

	return head
}

func translateHead(msg []byte) headc {
	var head headc
	head.seqN = binary.LittleEndian.Uint32(msg[0:4])
	head.ackN = binary.LittleEndian.Uint32(msg[4:8])
	head.connID = binary.LittleEndian.Uint16(msg[8:12])
	head.ack = uint(binary.LittleEndian.Uint16(msg[12:16]))
	head.syn = uint(binary.LittleEndian.Uint16(msg[16:20]))
	head.fin = uint(binary.LittleEndian.Uint16(msg[20:24]))

	return head
}

func createMsg(head headc) *bytes.Buffer {
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

func startTimer(time int) {

}

func tahoe() {
	if cwnd < ssthresh {
		cwnd += 512
	} else if cwnd >= ssthresh {
		cwnd += (512 * 512) / cwnd
	}
}

func closeConnection(udpConn *net.UDPConn, head headc) {
	// reply := make([]byte, maxDatagSize)
	// udpConn.ReadFrom(reply)
	// headAux := translateHead(reply)
	// if headAux.ack == 1 {
	/*Esperar por 2 segundos respondendo todas as mensagem T_T*/
	// }
	udpConn.Close()
	fmt.Println("File send") /*mudar a mensagem*/
	os.Exit(1)
}

func sendFile(filename string, udpConn *net.UDPConn, head headc, src *net.UDPAddr) {
	var (
		aux     = int64(0)
		headAux headc
		buf     *bytes.Buffer
	)

	msg := make([]byte, maxDatagSize)
	reply := make([]byte, maxDatagSize)
	file, _ := os.Open(filename)
	defer file.Close()

	buf = createMsg(head)
	num, _ := buf.Read(msg)
	buf.Reset()
	_, err := file.ReadAt(msg[num:], aux)
	fmt.Println("Sending ", num, " bytes")
	udpConn.WriteToUDP(msg, src)

	udpConn.ReadFromUDP(msg[0:])
	// _, udpAddr, err := udpConn.ReadFromUDP(msg[0:])
	// if err != nil {
	// 	log.Fatal(err)
	// 	os.Exit(-1)
	// }
	// conn, err := net.DialUDP("udp", nil, udpAddr)
	// if err != nil {
	// 	log.Fatal(err)
	// 	os.Exit(-1)
	// }
	// defer conn.Close()

	for {
		if head.ack == 1 {
			head.ack = 0
		} else {
			fmt.Println("Wrong package")
			os.Exit(-1)
		}

		buf = createMsg(head)
		num, _ = buf.Read(msg)
		buf.Reset()
		_, err = file.ReadAt(msg[num:], aux)

		if err == io.EOF {
			head.fin = 1
			// msg[20:24] = []byte(uint16(head.fin))
			fmt.Println("Sending package with FIN to server")
			udpConn.WriteToUDP(msg, src)
			file.Close()
			fmt.Println("Closing connection")
			closeConnection(udpConn, head)
		} else {
			fmt.Println("Sending ", head.ackN, " bytes")
			udpConn.WriteToUDP(msg, src)
			fmt.Println("Waiting package with ACK from server")
			udpConn.ReadFromUDP(reply)
			headAux = translateHead(reply)
			if headAux.ack == 1 {
				aux += payload
				head = headAux
			}
		}
	}
}

func handshake(udpAddr *net.UDPAddr, head headc, file string) {
	service := "127.0.0.1:0"
	udp, err := net.ResolveUDPAddr("udp", service)
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}

	// conn, err := net.DialUDP("udp", nil, udpAddr)
	conn, err := net.ListenUDP("udp", udp)
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}
	defer conn.Close()

	head.syn = 1
	buf := createMsg(head)
	msg := make([]byte, buf.Len())
	buf.Read(msg)
	fmt.Println("Sending package with SYN to server", udpAddr)
	conn.WriteToUDP(msg, udpAddr)

	reply := make([]byte, maxDatagSize)
	fmt.Println("Waiting package with ACK and SYN from server", udpAddr)
	_, src, _ := conn.ReadFromUDP(reply[0:])
	head = translateHead(reply)

	if head.syn == 1 && head.ack == 1 {
		fmt.Println("Recived package with ACK and SYN from server", udpAddr)
		head.syn = 0
		fmt.Println("Sending package with ACK to server", udpAddr)
		sendFile(file, conn, head, src)
	} else {
		fmt.Println("Handshake failed")
		os.Exit(-1)
	}
}

func handleConnection(addr string, file string) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		fmt.Println("Invalid address")
		os.Exit(-1)
	}

	head := createHead()
	handshake(udpAddr, head, file)
}

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Error invalid command")
		os.Exit(-1)
	}

	filename := os.Args[3]
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error invalid file")
		os.Exit(-1)
	}
	file.Close()

	hostname := os.Args[1]
	port := os.Args[2]
	addr := hostname
	addr += ":"
	addr += port

	handleConnection(addr, filename)
}
