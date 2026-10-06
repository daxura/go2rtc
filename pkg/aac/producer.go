package aac

import (
	"bufio"
	"errors"
	"io"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/pion/rtp"
)

type Producer struct {
	core.Connection
	rd *bufio.Reader
}

func Open(r io.Reader) (*Producer, error) {
	rd := bufio.NewReader(r)

	b, err := rd.Peek(ADTSHeaderSize)
	if err != nil {
		return nil, err
	}

	codec := ADTSToCodec(b)
	if codec == nil {
		return nil, errors.New("adts: wrong header")
	}
	codec.PayloadType = core.PayloadTypeRAW

	medias := []*core.Media{
		{
			Kind:      core.KindAudio,
			Direction: core.DirectionRecvonly,
			Codecs:    []*core.Codec{codec},
		},
	}
	return &Producer{
		Connection: core.Connection{
			ID:         core.NewID(),
			FormatName: "adts",
			Medias:     medias,
			Transport:  r,
		},
		rd: rd,
	}, nil
}

func (c *Producer) Start() error {
	for {
		// read ADTS header
		adts := make([]byte, ADTSHeaderSize)
		if _, err := io.ReadFull(c.rd, adts); err != nil {
			return err
		}

		frameSize := ReadADTSSize(adts)
		minSize := uint16(ADTSHeaderSize)
		if HasCRC(adts) {
			minSize += 2
		}
		// a corrupt header must not underflow the uint16 size (would read 64 KB of garbage and panic RTPPay)
		if !IsADTS(adts) || frameSize <= minSize {
			return errors.New("adts: wrong frame header")
		}
		auSize := frameSize - ADTSHeaderSize

		if HasCRC(adts) {
			// skip CRC after header
			if _, err := c.rd.Discard(2); err != nil {
				return err
			}
			auSize -= 2
		}

		// read AAC payload after header
		payload := make([]byte, auSize)
		if _, err := io.ReadFull(c.rd, payload); err != nil {
			return err
		}

		c.Recv += int(auSize)

		if len(c.Receivers) == 0 {
			continue
		}

		pkt := &rtp.Packet{
			Header:  rtp.Header{Timestamp: core.Now90000()},
			Payload: payload,
		}
		c.Receivers[0].WriteRTP(pkt)
	}
}
