package head

import (
	"bytes"
	"encoding/binary"
)

type HeadC struct {
	SeqN    uint32
	AckN    uint32
	ConnID  uint16
	NotUsed uint16
	Ack     uint
	Syn     uint
	Fin     uint
}

func CreateHeadC() HeadC {
	// Função para criar um header default
	var h HeadC

	h.SeqN = 12345
	h.AckN = 0
	h.ConnID = 0
	h.NotUsed = 0
	h.Ack = 0
	h.Syn = 0
	h.Fin = 0

	return h
}

func TranslateHeadC(msg []byte) HeadC {
	// Função para traduzir o header do pacote recebido
	var h HeadC

	// Converte os dados para o formato LittleEndian, com 32 btes
	h.SeqN = binary.LittleEndian.Uint32(msg[0:4])
	h.AckN = binary.LittleEndian.Uint32(msg[4:8])

	// Converte os dados para o formato LittleEndian, com 16 btes
	h.ConnID = binary.LittleEndian.Uint16(msg[8:12])
	h.Ack = uint(binary.LittleEndian.Uint16(msg[12:16]))
	h.Syn = uint(binary.LittleEndian.Uint16(msg[16:20]))
	h.Fin = uint(binary.LittleEndian.Uint16(msg[20:24]))

	return h
}

// Função para criar o header da mensagem
func CreateMsgC(h HeadC) *bytes.Buffer {
	// Array para armazenar a mensagem
	msg := make([]byte, 4)

	// Buffer para armazenar os bytes da mensagem
	buf := bytes.NewBuffer(nil)

	// Converte os dados do seqN para LittleEndian, com 32 btes, e armazena em msg
	binary.LittleEndian.PutUint32(msg, h.SeqN)

	// Escreve o conteudo de msg no buffer
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

	// retorna o buffer
	return buf
}
