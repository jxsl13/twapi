package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

var (
	errStartIndexIsNegative = errors.New("start index is negative")
	errEndIndexIsNegative   = errors.New("end index is negative")
	ErrNotACommand          = errors.New("not a config command")
	ErrIsComment            = errors.New("is a comment")
	ErrCommandListIsNil     = errors.New("command list is nil (uninitialized)")
)

// NewConfig initialized a new and empty command list
func NewConfig() Config {
	return make(Config, 0, 1)
}

// ParseConfig parses a config from a byte slice
func ParseConfigBytes(data []byte) (Config, error) {
	var c Config
	err := c.UnmarshalText(data)
	return c, err
}

// NewConfigFromReader parses a config from an io.Reader
func ParseConfigReader(r io.Reader) (Config, error) {
	var c Config
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = c.UnmarshalText(data)
	if err != nil {
		return nil, err
	}
	return c, nil
}

type Config []Command

func (cc Config) MarshalText() ([]byte, error) {
	var (
		buf = bytes.NewBuffer(make([]byte, 0, 64*len(cc)))
		err error
		txt []byte
	)

	for _, cmd := range cc {
		txt, err = cmd.MarshalText()
		if err != nil {
			return nil, err
		}
		buf.Write(txt)
		buf.WriteRune('\n')
	}

	return buf.Bytes(), nil
}

func (cc *Config) UnmarshalText(data []byte) error {
	if cc == nil {
		return ErrCommandListIsNil
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))

	commands := make([]Command, 0, 4)
	for scanner.Scan() {

		var cmd Command
		err := cmd.UnmarshalText(scanner.Bytes())
		if err == nil {
			commands = append(commands, cmd)
		} else if errors.Is(err, ErrNotACommand) {
			continue
		} else {
			return err
		}
	}
	*cc = commands
	return nil
}

type Command struct {
	Name string
	Args []string
}

func (c *Command) MarshalText() ([]byte, error) {
	size := len(c.Name)
	for _, arg := range c.Args {
		size += len(arg)
	}
	buf := bytes.NewBuffer(make([]byte, 0, size))

	buf.WriteString(c.Name)
	buf.WriteRune(' ')

	for idx, arg := range c.Args {
		buf.WriteString(strconv.Quote(arg))
		if idx < len(c.Args)-1 {
			buf.WriteRune(' ')
		}
	}

	return buf.Bytes(), nil
}

func (c *Command) UnmarshalText(data []byte) error {
	cmd, err := parseLine(data)
	if err != nil {
		return err
	}

	*c = cmd
	return nil
}

// parseLine parses a single config line
func parseLine(data []byte) (Command, error) {
	if len(data) < 3 {
		return Command{}, fmt.Errorf("%w: %v", ErrNotACommand, "invalid data length")
	}

	cmdStart := skipWhitespace(data)
	if cmdStart < 0 {
		return Command{}, fmt.Errorf("%w: %v", ErrNotACommand, errStartIndexIsNegative)
	}
	cmdEnd := cmdStart + skipToWhitespace(data[cmdStart:])
	if cmdEnd < 0 {
		return Command{}, fmt.Errorf("%w: %v", ErrNotACommand, errEndIndexIsNegative)
	}

	if data[cmdStart] == '#' {
		return Command{}, fmt.Errorf("%w: %w", ErrNotACommand, ErrIsComment)
	}

	command := Command{
		Name: string(data[cmdStart:cmdEnd]),
		Args: make([]string, 0, 1),
	}

	i := cmdEnd
outer:
	for i < len(data) {
		instr := 0

		j := i + skipWhitespace(data[i:])
		cmdStart = j

		for j < len(data) {
			c := data[j]

			if c == '"' {
				if j > 0 && data[j-1] != '\\' {

					if instr == 0 {
						cmdStart = j + 1
					} else {
						cmdEnd = j
						j++
						break
					}

					// no escape character before the ", valid "
					instr ^= 1
				}
			} else if instr == 0 {
				// no initial quote found
				if c == ';' {
					// command separator
					j++
				} else if c == '#' {
					// comment, no need to do anything more
					break outer
				} else if !isSpace(c) {
					// no initial quotes & word start
					j = j + skipToWhitespace(data[j:])
					cmdEnd = j
					break
				}
			}
			j++
			cmdEnd = j
		}

		if cmdStart < cmdEnd {
			arg := string(data[cmdStart:cmdEnd])
			command.Args = append(command.Args, arg)
		}

		i = j + 1
	}

	return command, nil
}
