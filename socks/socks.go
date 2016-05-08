package socks

import (
	"bufio"
	"encoding/binary"
	"fmt"
)

const (
	ResponseGranted = 0x5A
)

var (
	byteOrder = binary.BigEndian
)

type SocksHeader struct {
	Version uint8
	IP      [4]byte
	Port    uint16
	User    string
}

func ReadHeader(conn bufio.ReadWriter) (*SocksHeader, error) {
	var header SocksHeader
	binary.Read(&conn, byteOrder, &header.Version)
	conn.ReadByte()
	binary.Read(&conn, byteOrder, &header.Port)
	err := binary.Read(&conn, byteOrder, &header.IP)
	header.User, err = conn.ReadString(byte(0))

	if err != nil {
		return nil, err
	}

	return &header, nil
}

func WriteResponse(conn bufio.ReadWriter, responseCode byte) error {
	err := binary.Write(&conn, byteOrder, byte(0))
	err = binary.Write(&conn, byteOrder, &responseCode)
	err = binary.Write(&conn, byteOrder, uint16(0))
	err = binary.Write(&conn, byteOrder, uint32(0))
	return err
}

func (header *SocksHeader) IPAsString() string {
	return fmt.Sprintf("%d.%d.%d.%d", header.IP[0], header.IP[1], header.IP[2], header.IP[3])
}

func (header *SocksHeader) IPAndPortAsString() string {
	return fmt.Sprintf("%d.%d.%d.%d:%d", header.IP[0], header.IP[1], header.IP[2], header.IP[3], header.Port)
}
