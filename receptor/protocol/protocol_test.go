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
