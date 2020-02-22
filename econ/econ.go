package econ

import (
	"bytes"
	"errors"

	"github.com/reiver/go-telnet"
)

// Conn is the telnet connection to a teeworlds external console terminal(econ)
type Conn struct {
	*telnet.Conn
}

// ReadLine reads a line from the external console
func (conn *Conn) ReadLine() (string, error) {
	line := make([]byte, 0, 256)
	buffer := [1]byte{}

	for {
		n, err := conn.Read(buffer[:])
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}
		if buffer[0] == '\n' {
			buffer := [2]byte{0xFF, 0xFF} // explicitly initialize with non zero value

			// seemingly every line ends with two \x00\x00
			n, err = conn.Read(buffer[:])
			if err != nil {
				return "", err
			}
			if n != 2 || !bytes.Equal(buffer[:], []byte{0x00, 0x00}) {
				return "", errors.New("failed to read \\x00\\x00")
			}
			break
		}
		// n == 1 && buffer[0] != '\n'
		line = append(line, buffer[0])
	}
	return string(line), nil
}

// WriteLine writes a line to the external console and forces its execution by appending a \n
func (conn *Conn) WriteLine(line string) error {
	stream := []byte(line + "\n")

	for len(stream) > 0 {
		n, err := conn.Write(stream)

		if err != nil {
			return err
		}
		stream = stream[n:]
		if len(stream) == 0 {
			break
		}
	}
	return nil
}

// DialTo connects to a server's econ port and tries to log in.
func DialTo(address, password string) (*Conn, error) {
	telnetConn, err := telnet.DialTo(address)
	if err != nil {
		return nil, err
	}
	conn := &Conn{telnetConn}

	line, err := conn.ReadLine()
	if err != nil {
		conn.Close()
		return nil, err
	}

	if line != "Enter password:" {
		conn.Close()
		return nil, errors.New(line)
	}

	err = conn.WriteLine(password)
	if err != nil {
		conn.Close()
		return nil, err
	}

	line, err = conn.ReadLine()
	if err != nil {
		conn.Close()
		return nil, err
	}

	if line != "Authentication successful. External console access granted." {
		conn.Close()
		return nil, errors.New(line)
	}

	return conn, nil
}
