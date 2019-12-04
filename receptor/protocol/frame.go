package protocol

import (
	"bytes"
	"encoding/binary"
	//"encoding/json"
	"errors"
	"fmt"
	"io"
	//"strings"
	//"time"
)

var (
	errInvalidFrameType  = errors.New("invalid frame type")
	errFrameTooShort     = errors.New("frame too short")
	errFrameDataTooShort = errors.New("frame data too short")
)

type frameType int8

const (
	HeaderFrameType  frameType = 0
	PayloadFrameType frameType = 1
	CommandFrameType frameType = 2
)

type messageID [16]byte

func (mid messageID) String() string {
	return fmt.Sprintf("%X-%X-%X-%X-%X",
		mid[0:4],
		mid[4:6],
		mid[6:8],
		mid[8:10],
		mid[10:],
	)
}

const FrameHeaderLength int = 26

type FrameHeader struct {
	Type    frameType
	Version byte
	ID      uint32
	Length  uint32
	MsgID   messageID
}

func (f *FrameHeader) unmarshal(buf []byte) error {

	if len(buf) < FrameHeaderLength {
		return io.ErrUnexpectedEOF
	}

	// FIXME:  lots of buffer stuff here...is it really needed??
	fb := buf[0:FrameHeaderLength]

	r := bytes.NewReader(fb)

	err := binary.Read(r, binary.BigEndian, f)
	if err != nil {
		// FIXME: log the error
		fmt.Println("failed to read frame:", err)
		return err
	}

	fmt.Println("f:", f)

	if f.isValidType() != true {
		return errInvalidFrameType
	}

	return nil
}

func (f *FrameHeader) isValidType() bool {
	fmt.Println("f.Type: ", f.Type)
	return f.Type == HeaderFrameType || f.Type == PayloadFrameType || f.Type == CommandFrameType
}

func (f *FrameHeader) marshal() ([]byte, error) {
	w := new(bytes.Buffer)

	err := binary.Write(w, binary.BigEndian, f)
	if err != nil {
		fmt.Println("failed to write frame:", err)
		return nil, err
	}

	return w.Bytes(), nil
}

func readFrame(r io.Reader) (*FrameHeader, error) {
	buf := make([]byte, FrameHeaderLength)

	_, err := io.ReadFull(r, buf)
	if err != nil {
		// FIXME: log err
		return nil, errFrameTooShort
	}

	f := new(FrameHeader)

	err = f.unmarshal(buf)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func readFrameData(r io.Reader, dataLength uint32) ([]byte, error) {
	buf := make([]byte, dataLength)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		// FIXME: log err
		return nil, errFrameDataTooShort
	}

	return buf, nil
}

func parseFrameData(r io.Reader, t frameType, dataLength uint32) (Message, error) {

	buf, err := readFrameData(r, dataLength)
	if err != nil {
		// FIXME: log err
		return nil, err
	}

	var m Message
	switch t {
	case HeaderFrameType:
		fmt.Println("Header!")
		m = new(RoutingMessage)
	case PayloadFrameType:
		fmt.Println("Payload!")
		m = new(PayloadMessage)
	case CommandFrameType:
		fmt.Println("Command!")
		m, err = buildCommandMessage(buf)
	default:
		// FIXME: log the invalid type
		return nil, errInvalidFrameType
	}

	if err != nil {
		return nil, err
	}

	if err := m.unmarshal(buf); err != nil {
		fmt.Println("FIXME: unmarshal failed, err:", err)
		return nil, err
	}

	return m, nil
}
