package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"
)

var (
	errStartIndexIsNegative = errors.New("start index is negative")
	errEndIndexIsNegative   = errors.New("end index is negative")
	ErrNotACommand          = errors.New("not a config command")
	ErrIsComment            = errors.New("comment")
	ErrCommandListIsNil     = errors.New("command list is nil (uninitialized)")
)

// NewCommands initialized a new and empty command list
func NewCommands() Commands {
	return make(Commands, 0, 1)
}

type Commands []Command

func (cc *Commands) String() string {
	var sb strings.Builder
	sb.Grow(len(*cc) * 64)
	for _, cmd := range *cc {
		sb.WriteString(cmd.String())
		sb.WriteRune('\n')
	}

	return sb.String()
}

func (cc *Commands) UnmarshalBinary(data []byte) error {
	if cc == nil {
		return ErrCommandListIsNil
	}
	*cc = Parse(data)
	return nil
}

type Command struct {
	Name string
	Args []string
}

func (c *Command) String() string {
	var sb strings.Builder
	size := len(c.Name)
	for _, arg := range c.Args {
		size += len(arg)
	}
	sb.Grow(size)

	sb.WriteString(c.Name)
	sb.WriteRune(' ')

	for idx, arg := range c.Args {
		sb.WriteRune('"')
		sb.WriteString(arg)
		sb.WriteRune('"')
		if idx < len(c.Args)-1 {
			sb.WriteRune(' ')
		}
	}

	return sb.String()
}

func (c *Command) UnmarshalBinary(data []byte) error {
	cmd, err := ParseLine(data)
	if err != nil {
		return err
	}

	*c = *cmd
	return nil
}

// Parse parses a whole cfg file
// Redundant configuration lines stay part of the list
func Parse(data []byte) Commands {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	commands := make([]Command, 0, 16)
	for scanner.Scan() {
		cmd, err := ParseLine(scanner.Bytes())
		if err == nil && cmd != nil {
			commands = append(commands, *cmd)
		}
	}
	return commands
}

// ParseLine parses a single config line
func ParseLine(data []byte) (*Command, error) {
	if len(data) < 3 {
		return nil, fmt.Errorf("%w: %v", ErrNotACommand, "invalid data length")
	}

	cmdStart := skipWhitespace(data)
	if cmdStart < 0 {
		return nil, fmt.Errorf("%w: %v", ErrNotACommand, errStartIndexIsNegative)
	}
	cmdEnd := cmdStart + skipToWhitespace(data[cmdStart:])
	if cmdEnd < 0 {
		return nil, fmt.Errorf("%w: %v", ErrNotACommand, errEndIndexIsNegative)
	}

	if data[cmdStart] == '#' {
		return nil, ErrIsComment
	}

	command := &Command{
		Name: string(data[cmdStart:cmdEnd]),
		Args: make([]string, 0, 1),
	}

	i := cmdEnd
outer:
	for i < len(data) {
		instr := 0

		j := i + skipWhitespace(data[i:])
		cmdStart = j
		isStr := false

		for j < len(data) {
			c := data[j]

			if c == '"' {
				if j > 0 && data[j-1] != '\\' {

					if instr == 0 {
						cmdStart = j + 1
					} else {
						cmdEnd = j
						j++
						isStr = true
						break
					}

					// no escape character before the ", valid "
					instr ^= 1
				}
			} else if instr == 0 {
				if c == ';' {
					// command separator
					j++
				} else if c == '#' {
					// comment, no need to do anything more
					break outer
				}
			}
			j++
			cmdEnd = j
		}

		// strings may contain whitespaces for indentations
		arg := string(data[cmdStart:cmdEnd])
		if !isStr {
			arg = strings.TrimSpace(arg)
		}
		command.Args = append(command.Args, arg)
		i = j + 1
	}

	return command, nil
}
