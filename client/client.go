package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"../head"
)

const (
	maxDatagSize = 524
	payload      = 500
	maxSeq       = 102400
	maxRec       = 102400
	minCWND      = 512
	limSSTHRESH  = 10000
)

func closeConnection(udpConn *net.UDPConn, h head.HeadC) {
	// Variavel para receber a mensagem do servidor
	reply := make([]byte, maxDatagSize)

	// Espera a resposta do servidor
	udpConn.ReadFromUDP(reply)

	// Variavel para armazenar o header da mensagem
	hAux := head.TranslateHeadC(reply)

	// Verifica se o ACK da mensagem recebida está correto
	if hAux.Ack == 1 {
		// Responder mensagem do servidor por 2 segundos
	}

	// Finaliza a conexão com o servidor
	udpConn.Close()
	fmt.Println("File send")
	os.Exit(1)
}

func sendFile(filename string, udpConn *net.UDPConn, h head.HeadC, src *net.UDPAddr) {
	// Variaveis para armazenar a mensagem e a resposta do servidor respectivamente
	msg := make([]byte, maxDatagSize)
	reply := make([]byte, maxDatagSize)
	var aux = int64(0)

	// Abre o arquivo e armazena em file, não será necessario verificar se existe algum erro pois ja foi verificado anteriormente
	file, _ := os.Open(filename)

	// Altera o valor do buf para o header da mensagem
	buf := head.CreateMsgC(h)

	// Variavel num irá receber a quantidade de bytes lidos, não será necessario tratar o erro, portanto _ no lugar de err
	num, _ := buf.Read(msg)

	// Reseta o buffer
	buf.Reset()

	// Copia o conteudo do arquivo para mensagem, iniciando de aux, a quantidade de bytes copiados irá ser igual ao tamanho dado pela mensagem, err irá receber o erro, caso exista esse erro será o io.EOF, ou seja end of file
	n, err := file.ReadAt(msg[num:], aux)

	// aux irá incrementar com a quantidade de n, sendo ele a quantidade de bytes lidos anteriormente
	aux += int64(n)

	fmt.Println("Sending", num, "bytes")

	// Envia a mensagem com uma parte do arquivo para o servidor
	udpConn.WriteToUDP(msg, src)

	// Aguarda a resposta do servidor e, quando recebida, armazena em reply
	udpConn.ReadFromUDP(reply[0:])

	for {
		if h.Ack == 1 {
			h.Ack = 0
		} else {
			fmt.Println("Wrong package")
			os.Exit(0)
			// Fazer o tratamento para esse caso
		}

		buf = head.CreateMsgC(h)
		num, _ = buf.Read(msg[0:])
		buf.Reset()
		_, err = file.ReadAt(msg[num:], aux)

		if err == io.EOF {
			// Altera o valor do FIN no header
			h.Fin = 1

			// Atualiza o valor do FIN na mensagem
			binary.LittleEndian.PutUint16(msg[20:24], uint16(h.Fin))

			fmt.Println("Sending package with FIN to server", src)

			// Envia a mensagem com o FIN para o servidor
			udpConn.WriteToUDP(msg, src)

			// Fecha o arquivo
			file.Close()

			fmt.Println("Closing connection")
			closeConnection(udpConn, h)
		} else {
			fmt.Println("Sending", h.AckN, "bytes")

			// Envia a mensagem com parte do arquivo para o servidor
			udpConn.WriteToUDP(msg, src)

			fmt.Println("Waiting package with ACK from server")

			// Espera a resposta com o servidor
			udpConn.ReadFromUDP(reply)

			// Variavel para armazenar o header da mensagem do servidor
			hAux := head.TranslateHeadC(reply)

			// Verifica se o ACK da mensagem do servidor está correto
			if hAux.Ack == 1 {
				// Incrementa o aux, dessa forma é possivel contralar a posição do arquivo
				aux += payload
				h = hAux
			}
		}
	}
}

func handshake(udpAddr *net.UDPAddr, h head.HeadC, filename string) {
	// Pega o endereço do cliente, neste caso o endereço local
	service := "127.0.0.1:0"

	// Converte o endereço do cliente para um endereço UDP
	udp, err := net.ResolveUDPAddr("udp", service)

	// Verifica se existe algum erro na converção do endereço
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}

	// Cria uma conexão UDP
	conn, err := net.ListenUDP("udp", udp)

	// Verifica se existe algum erro na criação da conexão
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}

	// Defere o encerramento da conexão para quando encerrar a função
	defer conn.Close()

	h.Syn = 1

	// Buffer para armazenar a mensagem
	buf := head.CreateMsgC(h)

	// Variavel para armazenar a mensagem
	msg := make([]byte, buf.Len())

	// Passa o conteudo do buffer para a variavel
	buf.Read(msg)

	fmt.Println("Sending package with SYN to server", udpAddr)

	// Envia a mensagem para o servidor
	conn.WriteToUDP(msg, udpAddr)

	// Variavel para armazenar a resposta do servidor
	reply := make([]byte, maxDatagSize)

	fmt.Println("Waiting package with ACK and SYN from server", udpAddr)

	// Aguarda a resposta do servidor, armazena a resposta em reply e salva o endereço usado pelo servidor para enviar a resposta em src
	_, src, _ := conn.ReadFromUDP(reply[0:])

	// Altera o header do cliente com os valores da resposta do servidor
	h = head.TranslateHeadC(reply)

	// Verifica se o SYN e o ACK estão corretos, caso não estejam finaliza a execução do programa
	if h.Syn == 1 && h.Ack == 1 {
		fmt.Println("Recived package with ACK and SYN from server", udpAddr)

		h.Syn = 0

		fmt.Println("Sending package with ACK to server")

		sendFile(filename, conn, h, src)
	} else {
		fmt.Println("Handshake failed")
		os.Exit(0)
	}
}

func handleConnection(addr string, filename string) {
	// Converte o endereço dado para um endereço UDP
	udpAddr, err := net.ResolveUDPAddr("udp", addr)

	// Verifica se existe algum ao coverter o endereço
	if err != nil {
		fmt.Println("Invalid address")
		os.Exit(0)
	}

	// Cria o header do cliente
	h := head.CreateHeadC()

	// Inicia o handshake com o servidor
	handshake(udpAddr, h, filename)
}

func main() {
	// Verifica se o comando está correto
	if len(os.Args) != 4 {
		fmt.Println("Error invalid command")
		os.Exit(-1)
	}

	// Variavel para guardar o nome do arquivo
	filename := os.Args[3]

	// Open irá tentar abrir o arquivo dado, caso consiga o arquivo será salvo em file o err será vazio, caso contrario err irá receber o erro
	file, err := os.Open(filename)

	// Verifica se existe algum erro ao abrir o arquivo
	if err != nil {
		fmt.Println("Error invalid file")
		os.Exit(-1)
	}

	// Fechando o acesso ao aquivo, pois não será necessario usa-lo por enquanto
	file.Close()

	// Variavel para armazenar o endereço, pegando primeiro o endereço do servidor e em seguida a porta a ser utilizada
	addr := os.Args[1]
	addr += ":"
	addr += os.Args[2]

	handleConnection(addr, filename)
}
