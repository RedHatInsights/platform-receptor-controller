package protocol

import (
	"testing"
)

func TestHiMessageUnmarshal(t *testing.T) {
	//s := "{\"cmd\": \"HI\", \"id\": \"node_01\", \"expire_time\": \"2006-01-02T15:04:05+07:00\"}"
	s := "{\"cmd\": \"HI\", \"id\": \"node_01\", \"expire_time\": \"2015-01-16T16:52:58.547366+01:00\"}"

	message, err := ParseMessage([]byte(s))
	if err != nil {
		//t.Errorf("Message parsing failed, got: %d, want: %d.", minimum.Cost, expected_minimum)
		t.Errorf("Message parsing failed, err: %s", err)
	}
	t.Logf("msg:%s", message)

	if message.Type() != HiMessageType {
		t.Errorf("Message parsing failed, got: %d, want: %d.", message.Type(), HiMessageType)
	}
}

func TestFrameUnmarshal(t *testing.T) {
	b := []byte{01, // type
		22,                     // version
		0x00, 0x00, 0x00, 0x0f, // id
		0x00, 0x00, 0x00, 0x08, // length
		0x78, 0x56, 0x34, 0x12, 0x34, 0x12, 0x78, 0x56, // uuid hi
		0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, // uuid low
		0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78} // header/payload/etc

	f := frame{}
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

	t.Logf("frame msg:%d", f.Type)
	t.Logf("frame version:%d", f.Version)
	t.Logf("frame id:%d", f.ID)
	t.Logf("frame msgid: %X-%X-%X-%X-%X", f.MsgID[0:4], f.MsgID[4:6], f.MsgID[6:8], f.MsgID[8:10], f.MsgID[10:])
	t.Logf("frame %+v", f)
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
		f := frame{}
		err := f.unmarshal(buff)
		if err == nil {
			t.Fatalf("[%s] expected error, but none occurred", testName)
		}
	}
}

func TestFrameMarshal(t *testing.T) {
	f := frame{Type: 10, Version: 22, ID: 15, Length: 8}
	t.Logf("frame %+v", f)
	b, err := f.marshal()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("frame buffer:%v", b)
}
