package require

import (
	"bufio"
	"fmt"
	"strings"
)

func msgOrFmtMsg(msgAndArgs ...any) string {
	if len(msgAndArgs) == 0 || msgAndArgs == nil {
		return ""
	}
	if len(msgAndArgs) == 1 {
		msg := msgAndArgs[0]
		if msgAsStr, ok := msg.(string); ok {
			return msgAsStr
		}
		return fmt.Sprintf("%+v", msg)
	}
	if len(msgAndArgs) > 1 {
		return fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
	return ""
}

type labeledMessage struct {
	label   string
	message string
}

type labeledMessages []labeledMessage

func (lm labeledMessages) String() string {
	longestLabel := 0
	numLabels := len(lm)
	msgSizeTotal := 0
	for _, v := range lm {
		if len(v.label) > longestLabel {
			longestLabel = len(v.label)
		}
		msgSizeTotal += len(v.message)
	}

	var sb strings.Builder
	sb.Grow(msgSizeTotal + numLabels*(longestLabel+8))
	sb.WriteString("\n")

	for _, v := range lm {
		sb.WriteString("\t")
		sb.WriteString(v.label)
		sb.WriteString(":")
		sb.WriteString(strings.Repeat(" ", longestLabel-len(v.label)))
		sb.WriteString("\t")

		// indent lines
		for i, scanner := 0, bufio.NewScanner(strings.NewReader(v.message)); scanner.Scan(); i++ {
			// no need to align first line because it starts at the correct location (after the label)
			if i != 0 {
				// append alignLen+1 spaces to align with "{{longestLabel}}:" before adding tab
				sb.WriteString("\n\t")
				sb.WriteString(strings.Repeat(" ", longestLabel+1))
				sb.WriteString("\t")
			}
			// write line
			sb.WriteString(scanner.Text())
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
