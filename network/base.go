package network

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"github.com/jxsl13/twapi/compression"
	"github.com/jxsl13/twapi/protocol"
)

// NilNetBase is the uninitialized state of
var NilNetBase NetBase

type NetBase struct {
	ctx context.Context

	socket   *NetSocket
	huffmann *compression.Huffman

	timer        *time.Timer
	timerDrained bool

	// IOHANDLE m_DataLogSent;
	// IOHANDLE m_DataLogRecv;
	requestTokenBuf [protocol.NetTokenRequestDataSize]byte
}

func NewNetBase(ctx context.Context, socket *NetSocket) (NetBase, error) {
	h, err := compression.NewHuffman(protocol.FrequencyTable)
	if err != nil {
		return NilNetBase, fmt.Errorf("failed to create net base: %w", err)
	}
	return NetBase{
		ctx:      ctx,
		socket:   socket,
		huffmann: h,
	}, nil
}

func (b *NetBase) Close() error {
	if b.timer != nil {
		closeTimer(b.timer, &b.timerDrained)
	}

	if b.socket != nil {
		return b.socket.Close()
	}
	return nil
}

// Wait waits for the duration d or for the internal context to be closed.
// It returns an error in cas ethe internal context was closed
func (b *NetBase) Wait(d time.Duration) error {
	if b.timer == nil || b.ctx == nil {
		return nil
	}

	resetTimer(b.timer, d, &b.timerDrained)
	select {
	case <-b.ctx.Done():
		return b.ctx.Err()
	case <-b.timer.C:
		b.timerDrained = true
	}
	return nil
}

func (b *NetBase) SendCtrlMsg(addr netip.AddrPort, token protocol.Token, ack int, msg protocol.ControlMsg, extraData ...byte) error {
	// TODO:implement
	return nil
}

func (b *NetBase) SendCtrlMsgWithToken(
	addr netip.AddrPort,
	token protocol.Token,
	ack int,
	msg protocol.ControlMsg,
	myToken protocol.Token,
	extended bool,
) error {
	// TODO:implement
	return nil
}

func (b *NetBase) SendPacketConnless(addr netip.AddrPort, token, responseToken protocol.Token, data []byte) error {
	return nil
}

func (b *NetBase) SendPacket(addr netip.AddrPort, packet PacketConstruct) error {
	return nil
}

func (b *NetBase) UnpackPacket(add netip.AddrPort, data []byte) (PacketConstruct, error) {
	return NilPacketContruct, nil
}
