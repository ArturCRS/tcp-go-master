package head

import (
	"bytes"
	"encoding/binary"
	"net"
)

// Estrutura do header para o servidor
type HeadS struct {
	UdpAddr *net.UDPAddr
	Path    string
	SeqN    uint32
	AckN    uint32
	ConnID  uint16
	NotUsed uint16
	Ack     uint
	Syn     uint
	Fin     uint
}

// Funcão para criar um cliente com informações default
func CreateHeadS(src *net.UDPAddr) HeadS {
	var head HeadS

	head.UdpAddr = src
	head.SeqN = 4321
	head.AckN = 0
	head.ConnID = 0
	head.NotUsed = 0
	head.Ack = 0
	head.Syn = 0
	head.Fin = 0

	return head
}

// Função para traduzir a mensagem
func TranslateHeadS(msg []byte) HeadS {
	var head HeadS

	// Converte os dados para o formato LittleEndian, com 32 btes
	head.SeqN = binary.LittleEndian.Uint32(msg[0:4])
	head.AckN = binary.LittleEndian.Uint32(msg[4:8])

	// Converte os dados para o formato LittleEndian, com 16 btes
	head.ConnID = binary.LittleEndian.Uint16(msg[8:12])
	head.Ack = uint(binary.LittleEndian.Uint16(msg[12:16]))
	head.Syn = uint(binary.LittleEndian.Uint16(msg[16:20]))
	head.Fin = uint(binary.LittleEndian.Uint16(msg[20:24]))

	return head
}

// Função para criar o header da mensagem
func CreateMsgS(h HeadS) *bytes.Buffer {
	// Cria um array de bytes, com tamanho 16
	msg := make([]byte, 4)

	// Cria um buffer, PQ o nil?
	buf := bytes.NewBuffer(nil)

	// Converte os dados do seqN para LittleEndian, com 32 btes, e armazena em msg
	binary.LittleEndian.PutUint32(msg, h.SeqN)

	// Escreve os dados do array no buffer
	buf.Write(msg)

	binary.LittleEndian.PutUint32(msg, h.AckN)
	buf.Write(msg)

	// Converte os dados do seqN para LittleEndian, com 16 btes, e armazena em msg
	binary.LittleEndian.PutUint16(msg, h.ConnID)
	buf.Write(msg)

	binary.LittleEndian.PutUint16(msg, uint16(h.Ack))
	buf.Write(msg)

	binary.LittleEndian.PutUint16(msg, uint16(h.Syn))
	buf.Write(msg)

	binary.LittleEndian.PutUint16(msg, uint16(h.Fin))
	buf.Write(msg)

	// Retorna o buffer
	return buf
}
