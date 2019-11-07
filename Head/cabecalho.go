package Head

type head struct {
	seq_n   uint32
	ack_n   uint32
	conn_id uint16
	ack     uint
	syn     uint
	fin     uint
}
