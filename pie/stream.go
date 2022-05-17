package pie

import (
	"errors"
	"github.com/Pie-Messaging/core/pie/pb"
	"github.com/lucas-clemente/quic-go"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"io"
	"time"
)

type Stream struct {
	Stream      quic.Stream
	sendBuf     []byte
	recvBuf     []byte
	parseOffset int
	readOffset  int
}

func NewStream(stream quic.Stream, recvBufArg ...[]byte) *Stream {
	var recvBuf []byte
	if len(recvBufArg) == 0 || recvBufArg[0] == nil {
		recvBuf = make([]byte, MaxMessageLen)
	} else {
		recvBuf = recvBufArg[0]
	}
	return &Stream{
		Stream:  stream,
		sendBuf: make([]byte, MaxMessageLen),
		recvBuf: recvBuf,
	}
}

func (s *Stream) SendMessage(message *pb.NetMessage) error {
	data, err := proto.Marshal(message)
	if err != nil {
		Logger.Println("Failed to marshal message:", err)
		return err
	}
	Logger.Println("Sending message:", message.String()[:MinInt(500, len(message.String()))])
	if err = s.SendData(data, time.Time{}); err != nil {
		return err
	}
	return nil
}

func (s *Stream) SendData(data []byte, deadline time.Time) error {
	_ = s.Stream.SetWriteDeadline(deadline)
	defer func() {
		_ = s.Stream.SetWriteDeadline(time.Time{})
	}()
	s.sendBuf = protowire.AppendBytes(s.sendBuf[:0], data)
	msgLen := len(s.sendBuf)
	if msgLen > MaxMessageLen {
		return ErrMsgTooLong
	}
	sentLen := 0
	for sentLen < msgLen {
		n, err := s.Stream.Write(s.sendBuf[sentLen:msgLen])
		if err != nil {
			Logger.Println("Failed to write to stream:", err)
			return err
		}
		sentLen += n
	}
	return nil
}

func (s *Stream) RecvMessage(deadline time.Time) (*pb.NetMessage, error) {
	data, _, _, err := s.RecvData(deadline)
	if err != nil {
		return nil, err
	}
	message := &pb.NetMessage{}
	if err = proto.Unmarshal(data, message); err != nil {
		Logger.Println("Failed to unmarshal message:", err)
		return nil, err
	}
	Logger.Println("Received message:", message.String()[:MinInt(500, len(message.String()))])
	return message, nil
}

func (s *Stream) RecvData(deadline time.Time, noLog ...bool) ([]byte, int, int, error) {
	_ = s.Stream.SetReadDeadline(deadline)
	defer func() {
		_ = s.Stream.SetReadDeadline(time.Time{})
	}()
	for {
		start, end, err := s.parseMessage()
		if err != nil {
			if err == ErrProtoEOF || err == ErrEmptyMsg {
				n, err := s.Stream.Read(s.recvBuf[s.readOffset:])
				if err != nil {
					Log(noLog, "Failed to read from stream:", err)
					return nil, -1, -1, err
				}
				s.readOffset += n
				continue
			}
			Logger.Println("Failed to parse message:", err)
			return nil, -1, -1, err
		}
		return s.recvBuf[start:end], start, end, nil
	}
}

func (s *Stream) parseMessage() (int, int, error) {
	msgULen, n := protowire.ConsumeVarint(s.recvBuf[s.parseOffset:s.readOffset])
	if n < 0 {
		if errors.Is(protowire.ParseError(n), io.ErrUnexpectedEOF) {
			return -1, -1, ErrProtoEOF
		}
		return -1, -1, ErrInvalidMsg
	}
	msgLen := int(msgULen)
	if msgLen < 0 {
		return -1, -1, ErrInvalidMsg
	}
	if msgLen > MaxMessageLen {
		return -1, -1, ErrMsgTooLong
	}
	if msgLen > s.readOffset-s.parseOffset-n {
		return -1, -1, ErrProtoEOF
	}
	s.parseOffset += n
	if msgLen == 0 {
		return -1, -1, ErrEmptyMsg
	}
	s.parseOffset += msgLen
	return n, n + msgLen, nil
}

func (s *Stream) Close() {
	if err := s.Stream.Close(); err != nil {
		Logger.Println("Failed to close stream:", err)
	}
}
