package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
	//"testing/iotest"
)

func TestFrameUnmarshal(t *testing.T) {
	b := []byte{01, // type
		22,                     // version
		0x00, 0x00, 0x00, 0x0f, // id
		0x00, 0x00, 0x00, 0x08, // length
		0x78, 0x56, 0x34, 0x12, 0x34, 0x12, 0x78, 0x56, // uuid hi
		0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, // uuid low
		0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78} // header/payload/etc

	f := FrameHeader{}
	err := f.unmarshal(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.Type != 01 {
		t.Fatalf("Frame Type parsing failed, got: %d, want: %d", f.Type, 01)
	}

	if f.Version != 22 {
		t.Fatalf("Frame Version parsing failed, got: %d, want: %d", f.Version, 22)
	}

	if f.ID != 15 {
		t.Fatalf("Frame ID parsing failed, got: %d, want: %d", f.ID, 15)
	}

	var lengthWanted uint32 = 8 // 255
	if f.Length != lengthWanted {
		t.Fatalf("Frame ID parsing failed, got: %d, want: %d", f.Length, lengthWanted)
	}

	// FIXME: verify the msg id looks right!!
}

func TestFrameUnmarshalError(t *testing.T) {
	subTests := map[string][]byte{
		"short_buffer": []byte{01, 22},
		"invalid_type": []byte{10, // type
			22,                     // version
			0x00, 0x00, 0x00, 0x0f, // id
			0x00, 0x00, 0x00, 0x08, // length
			0x78, 0x56, 0x34, 0x12, 0x34, 0x12, 0x78, 0x56, // uuid hi
			0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, // uuid low
			0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}, // header/payload/etc
	}

	for testName, buff := range subTests {
		f := FrameHeader{}
		err := f.unmarshal(buff)
		if err == nil {
			t.Fatalf("[%s] expected error, but none occurred", testName)
		}
	}
}

func TestFrameMarshal(t *testing.T) {
	f := FrameHeader{Type: 10, Version: 22, ID: 15, Length: 8}
	b, err := f.marshal()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(b) != FrameHeaderLength {
		t.Fatalf("invalid frame buffer length")
	}
}

func TestReadMessageInvalidFrameType(t *testing.T) {
	f := FrameHeader{Type: 10, Version: 22, ID: 15, Length: 8}
	b, err := f.marshal()

	r := bytes.NewReader(b)
	message, err := readMessage(r)
	if message != nil || err != errInvalidFrameType {
		t.Fatalf("expected an invalid frame type error!!")
	}
}

func TestReadMessageShortFrame(t *testing.T) {
	badFrame := []byte{0x00, 0x01}
	r := bytes.NewReader(badFrame)
	message, err := readMessage(r)
	if message != nil || err != errFrameTooShort {
		t.Fatalf("expected an frame too short error!!")
	}
}

func TestReadMessageShortFrameData(t *testing.T) {
	b := generateFrameByteArray(CommandFrameType, 123, []byte{0x00, 0x01, 0x02})

	// Make the frame data too short
	b = b[:len(b)-2]

	r := bytes.NewReader(b)
	message, err := readMessage(r)
	if message != nil || err != errFrameDataTooShort {
		t.Fatalf("expected an frame data too short error!!")
	}
}

func TestCommandMessageHi(t *testing.T) {
	commandMessage := []byte("{\"cmd\": \"HI\", \"id\": \"node_01\", \"expire_time\": \"2015-01-16T16:52:58.547366+01:00\"}")

	b := generateFrameByteArray(CommandFrameType, 123, commandMessage)

	r := bytes.NewReader(b)
	message, _ := readMessage(r)
	if message.Type() != HiMessageType {
		t.Fatalf("incorrect message type")
	}

	hiMessage := message.(*HiMessage)
	if hiMessage.Command != "HI" {
		t.Fatalf("incorrect command")
	}
}

func TestCommandMessageRouteTable(t *testing.T) {
	commandMessage := []byte("{\"cmd\": \"ROUTE\", \"id\": \"node_01\", \"edges\": [[\"node-a\", \"node-b\", 1]], \"seen\": [\"node-a\", \"node-b\"]}")

	b := generateFrameByteArray(CommandFrameType, 123, commandMessage)

	r := bytes.NewReader(b)
	message, _ := readMessage(r)
	if message.Type() != RouteTableMessageType {
		t.Fatalf("incorrect message type")
	}

	routeTableMessage := message.(*RouteTableMessage)
	if routeTableMessage.Command != "ROUTE" {
		t.Fatalf("incorrect command")
	}
}

func TestCommandMessageInvalidMessages(t *testing.T) {

	subTests := map[string][]byte{
		"invalid_msg":     []byte("{\"cmd\": \"HI\", \"id\": \"node_01\", \"expire_time\": }"),
		"invalid_command": []byte("{\"cmd\": \"FREDFLINTSTONE\"}"),
	}

	for testName, msgBuff := range subTests {

		b := generateFrameByteArray(CommandFrameType, 123, msgBuff)

		r := bytes.NewReader(b)
		//readMessage(iotest.NewReadLogger("read_logger", iotest.OneByteReader(r)))
		message, err := readMessage(r)
		fmt.Println("message:", message)
		fmt.Println("err:", err)
		if message != nil && err == nil {
			t.Fatalf("[%s] invalid response...expected an error, got success", testName)
		}
	}
}

func generateFrameByteArray(t frameType, messageID int, payload []byte) []byte {
	version := byte(22)
	b := []byte{byte(t),
		version,
		0x00, 0x00, 0x00, 0x08, // id
		0x00, 0x00, 0x00, 0x51, // length
		0x78, 0x56, 0x34, 0x12, 0x34, 0x12, 0x78, 0x56, // uuid hi
		0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78} // uuid low

	binary.BigEndian.PutUint32(b[2:6], uint32(messageID))

	binary.BigEndian.PutUint32(b[6:10], uint32(len(payload)))

	b = append(b, payload...)

	return b
}

func TestHeaderAndPayload(t *testing.T) {
	routingMessage := []byte("{\"sender\": \"123\", \"recipient\": \"345\", \"route_list\": [\"678\"]}")

	b := generateFrameByteArray(HeaderFrameType, 123, routingMessage)

	fmt.Printf("type(b):%T\n", b)
	fmt.Println("b:", b)
	fmt.Printf("len(routingMessage):%d\n", len(routingMessage))

	payload := []byte("{\"message_id\": \"123\", \"raw_payload\": \"BLAH!BLAH!\"}")
	payloadHeader := generateFrameByteArray(PayloadFrameType, 123, payload)
	fmt.Printf("len(payload):%d\n", len(payload))

	b = append(b, payloadHeader...)

	r := bytes.NewReader(b)

	//message, err := readMessage(iotest.NewReadLogger("read_logger", iotest.OneByteReader(r)))

	message, err := readMessage(r)
	fmt.Println("message:", message)
	fmt.Println("err:", err)
	if message.Type() != PayloadMessageType {
		t.Fatalf("incorrect message type")
	}

	payloadMessage := message.(*PayloadMessage)
	if payloadMessage.RoutingInfo.Sender != "123" {
		t.Fatalf("incorrect sender")
	}
}

func TestHeaderAndPayloadWithShortPayloadRead(t *testing.T) {
	routingMessage := []byte("{\"sender\": \"1234\", \"recipient\": \"345\", \"route_list\": [\"678\"]}")

	b := generateFrameByteArray(HeaderFrameType, 123, routingMessage)

	payload := []byte("{\"message_id\": \"123\", \"raw_payload\": \"BLAH!BLAH!\"}")
	payloadHeader := generateFrameByteArray(PayloadFrameType, 123, payload)

	b = append(b, payloadHeader...)

	// make the payload data short
	b = b[:len(b)-2]

	r := bytes.NewReader(b)

	//message, err := readMessage(iotest.NewReadLogger("read_logger", iotest.OneByteReader(r)))

	message, err := readMessage(r)
	fmt.Println("message:", message)
	fmt.Println("err:", err)
	if message != nil || err != errFrameDataTooShort {
		t.Fatalf("expected an invalid message error!!")
	}
}

func TestHeaderFollowedByIncorrectFrame(t *testing.T) {
	routingMessage := []byte("{\"sender\": \"1234\", \"recipient\": \"345\", \"route_list\": [\"678\"]}")

	b := generateFrameByteArray(HeaderFrameType, 123, routingMessage)

	// Add a Command frame behind a Header frame
	payload := []byte("{\"cmd\": \"blah\", \"raw_payload\": \"BLAH!BLAH!\"}")
	invalidFrame := generateFrameByteArray(CommandFrameType, 123, payload)

	b = append(b, invalidFrame...)

	r := bytes.NewReader(b)

	message, err := readMessage(r)
	if message != nil || err != errInvalidMessage {
		t.Fatalf("expected an invalid message error!!")
	}
}
