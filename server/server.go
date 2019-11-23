package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"

	"../head"
)

const ( //Variaveis constantes
	maxDatagSize = 524
	payLoad      = 500
	maxSeq       = 102400
	maxRec       = 102400
	minCWND      = 512
	limSSTHRESH  = 10000
)

var (
	// Lista de clientes da aplicação
	list = []head.HeadS{}

	// Contador para indentificar a quantidade de clientes
	counter = 1
)

func saveFile(h head.HeadS, bFile []byte) {
	//Abre o arquivo para escrever o conteudo da mensagem
	file, err := os.OpenFile(h.Path, os.O_APPEND|os.O_WRONLY, 0600)
	/*PS: os.O_APPEND é usado para dizer que estará adicionando dados ao arquivo
	  os.O_WRONLY é usado para abrir o arquivo somente para leitura
	  0600 é uma permissão, onde somente o dono do arquivo pode ler ou escrever
	*/

	// Verifica se existe algum erro ao abrir o arquivo
	if err != nil {
		//Tratar essa parte
	}

	// Escreve o conteudo da mensagem no final do arquivo, isso é possivel pois foi informado que iriamos adicionar dados no arquivo ao abri-lo
	file.Write(bFile)

	// Fecha o arquivo
	file.Close()

	// Finaliza a função
	return
}

func updateList(h head.HeadS) []head.HeadS {
	// Variavel auxiliar para receber os elementos da lista, menos a elemento a ser removido
	var l []head.HeadS

	// For para percorrer toda a lista
	for _, v := range list {

		// Verifica se o ID atual é igual ao ID do cliente a ser removido
		if v.ConnID == h.ConnID {
			// Caso o ID seja o mesmo pula para o próximo elemento
			continue
		} else {
			// Caso contrario o elemento é adicionado a lista auxiliar
			l = append(l, v)
		}
	}

	// Retorna a lista auxiliar
	return l
}

func closeConnection(udpConn *net.UDPConn, h head.HeadS) {
	// Array para guardar o header da mensagem para o usuário
	send := make([]byte, maxDatagSize)
	h.Ack = 1

	// Buffer para pegar a mensagem
	buf := head.CreateMsgS(h)

	// Transferindo a mensagem do buffer para o array, e em seguida resetando o buffer
	buf.Read(send)
	buf.Reset()

	// Enviando a mensagem para o usuáiro
	udpConn.WriteToUDP(send, h.UdpAddr)

	h.Ack = 0
	h.Fin = 0
	h.AckN = 0

	// Looping para enviar a mensagem para o usuário, e em seguida ficar esperando a resposta com ACK = 1, caso seja igual irá encerrar a conexão, caso contrario irá continuar enviando a mensagem até receber a resposta desejada
	for {
		buf = head.CreateMsgS(h)

		buf.Read(send)
		buf.Reset()

		// Envia o pacote para o cliente
		udpConn.WriteToUDP(send, h.UdpAddr)

		// Espera a resposta do cliente
		udpConn.ReadFromUDP(send)

		// Variavel usada para armazenar o header da mensagem recebida do cliente
		hAux := head.TranslateHeadS(send[0:24])

		// Verifica se o ACK do pacote recebido é igual a 1, se for encerra a conexão
		if hAux.Ack == 1 {
			// Encerra a conexão
			udpConn.Close()

			// Atualiza a lista de clientes
			list = updateList(h)

			return
		}
	}
}

func handleClient(msg []byte, h head.HeadS) {
	// Cria uma conexão com o cliente e da um bind
	conn, err := net.DialUDP("udp", nil, h.UdpAddr)

	// Verifica se existe algum erro ao esbalecer a conexão
	if err != nil {
		log.Fatal(err)
		return
	}

	// Defere o encerramento da conexão para quando encerrar a função
	defer conn.Close()

	// Envia o ultimo pacote do handshake
	conn.WriteToUDP(msg, h.UdpAddr)

	// Variavel para receber a resposta do cliente
	reply := make([]byte, maxDatagSize)

	// Variavel para enviar a resposta para o cliente
	send := make([]byte, maxDatagSize)
	// PS: Caso queira é possivel usar somente uma variavel para receber e enviar mensagens

	// Looping para ficar recebendo as mensagens do cliente
	for {
		// Espera a resposta do cliente, e armazena em reply
		fmt.Println("Waiting package from client", h.UdpAddr)
		conn.ReadFromUDP(reply)

		// Variavel auxiliar, para facilitar a identificação do cliente
		hAux := head.TranslateHeadS(reply[0:24])
		// PS: A partir de agora as mensagens terão além do header um conteudo, portanto é necessário separa-los, sendo os 24 primeiros bytes o header do cliente

		// Verifica se a mensagem recebida é o cliente desta thread
		if hAux.ConnID == h.ConnID {
			fmt.Println("Recived package from client", h.UdpAddr)

			// Modifica o valor do ACK, caso o ACK recebido seja 0
			if hAux.Ack == 0 {
				hAux.Ack = 1
			}

			// Variavel usada para armazenar o conteudo do arquivo
			file := reply[24:]

			// Veridica se o valor do FIN é 1, caso seja significa que o cliente não irá enviar mais pacotes, portanto é necessário fechar a conexão
			if hAux.Fin == 1 {
				h.Fin = 1
				saveFile(h, file)
				fmt.Println("Closing connection with client", h.UdpAddr)
				closeConnection(conn, h)
				return
			}

			// Verifica se o valor do ACKN recebido é o mesmo que o valor SEQ(numero de sequencia)
			if hAux.AckN == h.SeqN {
				saveFile(hAux, file)

				// Incrementa o ACKN do cliente e altera o ACK
				h.AckN = hAux.SeqN + uint32(payLoad)
				h.Ack = 1

				// Cria buffer para o header e em seguida passa o conteudo para send
				buf := head.CreateMsgS(h)
				buf.Read(send)

				// Limpa o buffer
				buf.Reset()

				fmt.Println("Sending ACKN", h.AckN, "to client", h.UdpAddr)
				conn.WriteToUDP(send, h.UdpAddr)
			} else {
				fmt.Println("Wrong package from client", h.UdpAddr)

				// Cria um buffer para o header e em seguida passa o conteudo do buffer para send
				buf := head.CreateMsgS(h)
				buf.Read(send)

				// Limpa o buffer
				buf.Reset()

				// Envia o pacote para o cliente
				conn.WriteToUDP(send, h.UdpAddr)
			}
		}
	}
}

func handshake(udpAddr *net.UDPAddr, msg []byte) {
	// Porta temporaria, usada somente para realizar o handshake
	service := "224.0.0.1:0"

	// Transforma a string para um endereço UDP
	udp, err := net.ResolveUDPAddr("udp", service)

	// Verifica se ocorreu algum erro na converção
	if err != nil {
		log.Fatal(err)
		fmt.Println("Failed to create a udpaddr from client", udpAddr)
		return
	}

	// Cria um array de bytes com tamanho de 524 bytes, usado para receber as mensagens do cliente
	reply := make([]byte, maxDatagSize)

	// Chamada da função para transformar a mensagem em informações do cliente
	h := head.TranslateHeadS(msg)

	fmt.Println("Recived package with SYN from client", udpAddr)

	// Verifica se o SYN da mensagem recebida é 1
	if h.Syn == uint(1) {
		// Cria uma conexão UDP no endereço fornecido
		conn, err := net.ListenUDP("udp", udp)

		// Verifica se existe algum erro na criação da conexão
		if err != nil {
			log.Fatal(err)
			fmt.Println("Falied to create connection")
			return
		}

		// difere o encerramento da conexão para quando encerrar a função
		defer conn.Close()

		// Atribui o ACK como 1
		h.Ack = 1

		// Atribui o ID do cliente com o valor do contador
		h.ConnID = uint16(counter)

		// Incrementa o contador
		counter++

		// Variavel que recebe a mensagem criada, sendo ele um buffer
		buf := head.CreateMsgS(h)

		// Cria um array com o mesmo tamanho do buffer
		send := make([]byte, buf.Len())

		// Escreve o conteudo do buffer na variavel send, todos os dados do buffer serão transferidos para o send
		buf.Read(send)

		fmt.Println("Sending package with ACK and SYN to client", udpAddr)

		// Envia as informações no send para o cliente com o endereço udpAddr, usando o UDP
		conn.WriteToUDP(send, udpAddr)

		fmt.Println("Waiting package with ACK from client", udpAddr)

		// Espera a resposta do cliente, armazenando a informação em reply
		conn.ReadFromUDP(reply)

		// traduz a mensagem recebida para a estrutura definida para o usuário
		h = head.TranslateHeadS(reply)

		if h.Ack == 1 {
			fmt.Println("Recived package with with ACK from client", udpAddr)

			// Salvando o endereço do cliente
			h.UdpAddr = udpAddr
			fmt.Println("Handshake with client", h.UdpAddr, "was succeful")

			// Salvando o caminho para armazenar os dados do arquivo
			h.Path = fmt.Sprint(h.ConnID)
			h.Path += ".txt"

			// Cria o arquivo para armazenar os dados recebidos do usuário
			os.Create(h.Path)

			// Cria uma variavel para armazenar os dados da mensagem
			file := reply[24:]

			// Escreve o conteudo da variavel no arquivo, PQ o 0666
			ioutil.WriteFile(h.Path, file, 0666)

			// Adiciona o cliente na lista
			list = append(list, h)

			// Cria o header da mensagem, e em seguida escreve o conteudo na variavel send
			buf = head.CreateMsgS(h)
			buf.Read(send[0:])

			fmt.Println("Sending package with ACK", h.AckN, "to client", h.UdpAddr)

			conn.Close()
			handleClient(send, h)
		} else {
			fmt.Println("Handshake with client", udpAddr, "failed")
			return
		}
	} else {
		fmt.Println("Handshake with client", udpAddr, "failed")
		return
	}
}

func serverMulticast(port string) {
	service := "224.0.0.1:"
	service += port
	// Transforma o endereço dado para um endereço em UDP
	addr, err := net.ResolveUDPAddr("udp", service)

	// Verifica se ouve algum erro apos a conversão
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}

	// Cria o socket para ficar esperando algum cliente
	listener, err := net.ListenMulticastUDP("udp", nil, addr)

	if err != nil {
		// Verifica se existe algum erro na criação do listener
		log.Fatal(err)
		os.Exit(-1)
	}

	// Cria um buffer para receber pedidos do cliente
	listener.SetReadBuffer(maxDatagSize)

	defer listener.Close()

	// Looping para ficar recebendo pedidos de clientes
	for {
		// Cria um array de bytes para a mensagem do cliente, com tamanho 534 bytes
		msg := make([]byte, maxDatagSize)

		// Recebe mensagem de clientes, guarda a mensagem recebida na variavel msg e o endereço do cliente em src
		_, src, _ := listener.ReadFromUDP(msg)

		fmt.Println("Iniciating handshake with client at", src)

		// Inicia a thread para fazer o handshake com o cliente
		go handshake(src, msg)
	}
}

func main() {
	// Verificar se o comando para iniciar está correto
	if len(os.Args) != 3 {
		fmt.Println("Error invalid command")
		os.Exit(-1)
	}

	// Caminho para o diretorio de arquivos
	fileDir := os.Args[2]

	// Verifica se é possivel alterar o diretorio atual para o diretorio expecificado, caso não não possa retorna um erro, caso contrario retorna vazio(nil)
	err := os.Chdir(fileDir)

	// Verifica  se existe algum erro ao mover para o diretorio informado
	if err != nil {
		// Variavel para receber o input do usuário
		var input string

		fmt.Println("Directory don't exist, do you wish to create a new directory with the given name?")
		fmt.Println("type 'y' for yes and 'n' for no")
		// Pegar a entrada do usuário
		_, err := fmt.Scanln(input)

		if input == "y" || err != nil {
			os.Mkdir(fileDir, os.ModeDir) //Cria o diretorio
			fmt.Println("Directory created successfuly")
			os.Chdir(fileDir) // Move para o diretorio
		} else {
			fmt.Println("Closing application")
			// Finaliza a aplicação
			os.Exit(0)
		}
	}

	fmt.Println("Application started")
	// Variavel para receber a porta que será usada
	port := os.Args[1]

	// Chamada da função para iniciar o servidor
	serverMulticast(port)
}
