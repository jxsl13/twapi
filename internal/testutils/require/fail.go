package require

import (
	"strings"
	"testing"
)

func FailNow(t *testing.T, errMsg string, msgAndArgs ...any) {
	t.Helper()

	labeledMessages := labeledMessages{
		{
			label:   "Error Trace",
			message: strings.Join(CallStack(), "\n\t\t\t"),
		},
		{
			label:   "Error",
			message: errMsg,
		},
	}

	message := msgOrFmtMsg(msgAndArgs...)
	if len(message) > 0 {
		labeledMessages = append(labeledMessages, labeledMessage{
			label:   "Message",
			message: message,
		})
	}

	t.Fatal(labeledMessages.String())
}
