package econ

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/reiver/go-telnet"
)

var (
	// ErrNetwork is returned when some network related error occurrs
	ErrNetwork = errors.New("a network error occurred")
	// ErrInvalidPassword is returned when the passed password is incorrect and does not grant access
	ErrInvalidPassword = errors.New("invalid password")
)

// Conn is the telnet connection to a teeworlds external console terminal(econ)
type Conn struct {
	telnetConn       *telnet.Conn
	address          string
	password         string
	reconnectRetries int
	reconnectDelay   time.Duration
	isClosed         bool
	closeOnce        sync.Once
}

// Close must be called when the connection is to be quit
func (c *Conn) Close() error {
	err := error(nil)
	c.closeOnce.Do(func() {
		c.logout()
		c.isClosed = true
		err = c.telnetConn.Close()
	})
	return err
}

func (c *Conn) logout() error {
	return c.unguardedWriteLine("logout")
}

// ReadLine reads a line from the external console
// if the connection is lost, it attempts to reconnect multiple times before
// trying to read the line again.
func (c *Conn) ReadLine() (string, error) {

	line, err := c.unguardedReadLine()
	if err == nil {
		// line read successfully
		return line, nil
	}
	// failed to get line
	recErr := c.reconnect()
	if recErr != nil {
		return "", fmt.Errorf("%w: %v", err, recErr)
	}
	return c.unguardedReadLine()
}

// no reconnect mechanisms guard this line reading
func (c *Conn) unguardedReadLine() (string, error) {
	stackArray := [256]byte{}
	stackArraySlice := stackArray[:0]
	lineBuffer := bytes.NewBuffer(stackArraySlice)

	singleCharBuffer := [1]byte{}
	singleCharBufferSlice := singleCharBuffer[:]

	// we read single byte arrays until we hit a linebreak
	for {
		n, err := c.telnetConn.Read(singleCharBufferSlice)
		if err != nil {
			return "", fmt.Errorf("%w:%v", ErrNetwork, err)
		}
		// failed to read one byte
		if n == 0 {
			continue
		}
		// we do hit a linebreak
		// we expect the next two characters to be 0xFF
		if singleCharBuffer[0] == '\n' {
			buffer := [2]byte{0xFF, 0xFF} // explicitly initialize with non zero value
			bufferSlice := buffer[:]

			// seemingly every line ends with two \x00\x00
			n, err = c.telnetConn.Read(bufferSlice)
			if err != nil {
				return "", fmt.Errorf("%w:%v", ErrNetwork, err)
			}
			if n != 2 || !bytes.Equal(bufferSlice, []byte{0x00, 0x00}) {
				return "", errors.New("failed to read \\x00\\x00")
			}

			// successfully got the two 0x00,
			// no need to append newline characters here
			break
		}
		// n == 1 && buffer[0] != '\n'
		lineBuffer.WriteByte(singleCharBuffer[0])
	}

	return lineBuffer.String(), nil
}

// WriteLine writes a line to the external console and forces its execution by appending a \n
func (c *Conn) WriteLine(line string) error {

	err := c.unguardedWriteLine(line)
	if err == nil {
		return nil
	}
	recErr := c.reconnect()
	if recErr != nil {
		return fmt.Errorf("%w: %v", err, recErr)
	}
	return c.unguardedWriteLine(line)
}

// WriteLine writes a line to the external console and forces its execution by appending a \n
func (c *Conn) unguardedWriteLine(line string) error {
	stream := []byte(line + "\n")

	for len(stream) > 0 {
		n, err := c.telnetConn.Write(stream)
		if err != nil {
			return fmt.Errorf("%w:%v", ErrNetwork, err)
		}
		stream = stream[n:]
		if len(stream) == 0 {
			break
		}
	}
	return nil
}

func (c *Conn) reconnect() error {

	c.logout()
	c.telnetConn.Close() // ignore possible error

	reconnectRetries := c.reconnectRetries
	retryAgain := func() bool {
		// finite reconnectRetries
		if c.reconnectRetries >= 0 {
			reconnectRetries--
			// abort retrying if connection is closed
			return !c.isClosed && reconnectRetries > 0
		}
		// infinite retries
		return !c.isClosed
	}

	// keep track of the last error that was returned
	err := error(nil)
	for retryAgain() {

		// reconnect tcp connection
		var telnetConn *telnet.Conn
		telnetConn, err = telnet.DialTo(c.address)
		if err != nil {
			time.Sleep(c.reconnectDelay)
			continue
		}
		// update internal state
		c.telnetConn = telnetConn
		err = c.authenticate()
		// network error -> reconnect makes sense
		if err != nil && errors.Is(err, ErrNetwork) {
			time.Sleep(c.reconnectDelay)
			continue
		} else if err != nil && errors.Is(err, ErrInvalidPassword) {
			// failed to authenticate -> reconnect makes no sense
			return err
		}
		// err must be nil here
		break
	}

	// retryAgain returns false: err != nil
	// early break: err == nil
	return err
}

// authenticate in the external console
func (c *Conn) authenticate() error {
	password := c.password

	line, err := c.unguardedReadLine()
	if err != nil {
		// forward network error
		return err
	}

	if line != "Enter password:" {
		return fmt.Errorf("authentication error, could not find password request line: %s", line)
	}

	err = c.unguardedWriteLine(password)
	if err != nil {
		// forward network error
		return err
	}

	line, err = c.unguardedReadLine()
	if err != nil {
		// forward network error
		return err
	}

	if line != "Authentication successful. External console access granted." {
		return fmt.Errorf("%w: %s", ErrInvalidPassword, line)
	}
	return nil
}

// New creates a new econ connection that can be used to write or read lines from
// the teeworlds server via the external console. (The New function is a wrapper around DialTo)
// address is the <IP>:<PORT(ec_port)> address
// the password is the one you set via: ec_password
// You may want to decrease the ec_auth_timeout in order to get disconnected faster and not to block
// any of the 4 existing econ slots.
// You can also set your ec_bantime to anything other than 0 in order to ban people that try to connect to you external console and try incorrect credentials
// ec_output_level [1,2] allows to increase the logging level of your external console. This allows for more verbose econ output parsing
func New(address, password string) (*Conn, error) {
	return DialTo(address, password)
}

// DialTo connects to a server's econ port and tries to log in.
func DialTo(address, password string) (*Conn, error) {
	telnetConn, err := telnet.DialTo(address)
	if err != nil {
		return nil, err
	}
	c := &Conn{
		telnetConn:       telnetConn,
		address:          address,
		password:         password,
		reconnectDelay:   time.Second,
		reconnectRetries: 360, // retry 6 minutes before failing
	}

	err = c.authenticate()
	if err != nil {
		c.Close()
		return nil, err
	}

	return c, nil
}
