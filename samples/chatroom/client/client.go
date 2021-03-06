package main

import (
	"errors"
	"fmt"
	"log"
	"unsafe"

	. "github.com/ecofast/rtl/sysutils"
	"tcpsock.v2"
	"tcpsock.v2/samples/chatroom/protocol"
)

type client struct {
	*tcpsock.TcpClient
	recvBuf    []byte
	recvBufLen int
	roomID     uint8
	seatID     uint8
}

func (self *client) SockHandle() uint64 {
	return self.ID()
}

func (self *client) onConnect(c *tcpsock.TcpConn) tcpsock.TcpSession {
	log.Println("successfully connect to server", c.RawConn().RemoteAddr().String())
	return self
}

func (self *client) onDisconnect(c *tcpsock.TcpConn) {
	log.Println("disconnect from server", c.RawConn().RemoteAddr().String())
}

func (self *client) Read(b []byte) (n int, err error) {
	count := len(b)
	if count+self.recvBufLen > tcpsock.RecvBufLenMax {
		return 0, errors.New("invalid data")
	}

	self.recvBuf = append(self.recvBuf, b[0:count]...)
	self.recvBufLen += count
	offsize := 0
	offset := 0
	var head protocol.PacketHead
	for self.recvBufLen-offsize > protocol.SizeOfPacketHead {
		offset = 0
		head.Len = uint16(uint16(self.recvBuf[offsize+1])<<8 | uint16(self.recvBuf[offsize+0]))
		pkglen := int(protocol.SizeOfPacketHead + head.Len)
		if pkglen >= tcpsock.RecvBufLenMax {
			offsize = self.recvBufLen
			break
		}
		if offsize+pkglen > self.recvBufLen {
			break
		}
		offset += protocol.SizeOfPacketHeadLen
		head.Cmd = uint16(uint16(self.recvBuf[offsize+offset+1])<<8 | uint16(self.recvBuf[offsize+offset+0]))
		switch head.Cmd {
		case protocol.PT_NORMAL:
			offset += protocol.SizeOfPacketHeadCmd
			self.process(self.recvBuf[offsize+offset : offsize+offset+int(head.Len)])
		default:
			//
		}
		offsize += pkglen
	}

	self.recvBufLen -= offsize
	if self.recvBufLen > 0 {
		self.recvBuf = self.recvBuf[offsize : offsize+self.recvBufLen]
	} else {
		self.recvBuf = nil
	}
	return len(b), nil
}

func (self *client) process(b []byte) {
	switch BytesToUInt16(b[:protocol.SizeOfMsgHeadProtoID]) {
	case protocol.SM_PING:
		log.Println("[SM_PING]")
	case protocol.SM_IDENTITY:
		log.Println("[SM_IDENTITY]")
	case protocol.SM_REQROOMLIST:
		buf := b[protocol.SizeOfMsgHead:]
		s := ""
		for i := range buf {
			if s == "" {
				s = IntToStr(int(buf[i]))
				continue
			}
			s = s + "," + IntToStr(int(buf[i]))
		}
		log.Printf("[SM_REQROOMLIST] %s\n", s)
	case protocol.SM_ENTERROOM:
		param := BytesToUInt16(b[protocol.SizeOfMsgHeadProtoID : protocol.SizeOfMsgHeadProtoID+protocol.SizeOfMsgHeadParam])
		if param != 0xFFFF {
			self.roomID = uint8(param >> 8)
			self.seatID = uint8(param)
			log.Printf("[SM_ENTERROOM] RoomID:%d, SeatID:%d\n", self.roomID, self.seatID)
			return
		}
		log.Println("[SM_ENTERROOM] Fail")
	case protocol.SM_EXITROOM:
		param := BytesToUInt16(b[protocol.SizeOfMsgHeadProtoID : protocol.SizeOfMsgHeadProtoID+protocol.SizeOfMsgHeadParam])
		if param != 0xFFFF {
			log.Println("[SM_EXITROOM] OK")
			return
		}
		log.Println("[SM_EXITROOM] Fail")
	case protocol.SM_CHAT:
		name := bytes2str(b[protocol.SizeOfMsgHead : protocol.SizeOfMsgHead+protocol.SizeOfUserName])
		txt := bytes2str(b[protocol.SizeOfMsgHead+protocol.SizeOfUserName:])
		log.Printf("[SM_CHAT] %s: %s\n", name, txt)
	case protocol.SM_NOTIFY:
		log.Printf("[SM_NOTIFY] %s\n", bytes2str(b[protocol.SizeOfMsgHead:]))
	default:
		fmt.Println("?????")
	}
}

func bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func newTcpClient(addr string) *client {
	c := &client{
		roomID: 0xFF,
		seatID: 0xFF,
	}
	c.TcpClient = tcpsock.NewTcpClient(addr, c.onConnect, c.onDisconnect)
	return c
}
