package browser

// NewRequestPacket creates a new request payload from a token and a request header
func NewRequestPacket(t Token, requestHeader string) ([]byte, error) {
	req := RequestPacket{
		Token:  t,
		Header: requestHeader,
	}
	return req.MarshalBinary()
}

// RequestPacket is a packet that can be sent either to the master servers or to a game server
// in order to fetch data
type RequestPacket struct {
	Token Token
	// Header is the request header defining the type of the request
	Header string
}

// MarshalBinary packages the request into its binary representation
func (rp *RequestPacket) MarshalBinary() ([]byte, error) {
	tokenHeader, err := rp.Token.MarshalBinary()
	if err != nil {
		return nil, err
	}
	payload := make([]byte, 0, len(tokenHeader)+len(rp.Header))
	payload = append(payload, tokenHeader...)
	payload = append(payload, rp.Header...)
	return payload, nil
}
