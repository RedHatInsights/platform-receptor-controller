package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
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
		log.Println("failed to read frame:", err)
		return err
	}

	if f.isValidType() != true {
		return errInvalidFrameType
	}

	return nil
}

func (f *FrameHeader) isValidType() bool {
	return f.Type == HeaderFrameType || f.Type == PayloadFrameType || f.Type == CommandFrameType
}

func (f *FrameHeader) marshal() ([]byte, error) {
	w := new(bytes.Buffer)

	err := binary.Write(w, binary.BigEndian, f)
	if err != nil {
		log.Println("failed to write frame:", err)
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
		m = new(RoutingMessage)
	case PayloadFrameType:
		m = new(PayloadMessage)
	case CommandFrameType:
		m, err = buildCommandMessage(buf)
	default:
		// FIXME: log the invalid type
		return nil, errInvalidFrameType
	}

	if err != nil {
		return nil, err
	}

	if err := m.unmarshal(buf); err != nil {
		log.Println("FIXME: unmarshal failed, err:", err)
		return nil, err
	}

	return m, nil
}

func writeFrame(w io.Writer, ftype frameType, frameData []byte) error {

	frameHeader := FrameHeader{Version: 1, ID: 1} // FIXME: pass this in
	frameHeader.Type = ftype
	frameHeader.Length = uint32(len(frameData))

	frameHeaderBuffer, err := frameHeader.marshal()
	if err != nil {
		// FIXME: log the error
		log.Println("error marshalling frame header")
		return err
	}

	n, err := w.Write(frameHeaderBuffer)
	if n != len(frameHeaderBuffer) || err != nil {
		// FIXME: log the error
		log.Println("error writing frame header")
		return err
	}

	n, err = w.Write(frameData)
	if n != len(frameData) || err != nil {
		// FIXME: log the error
		log.Println("error writing frame data")
		return err
	}

	return nil
}
