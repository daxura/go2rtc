package h265

import (
	"encoding/binary"
	"testing"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/pion/rtp"
)

// A sender that ends each access unit with a small SEI carrying the RTP marker
// (go2rtc's Annex-B producer + RTPPay) must still produce access units.
func TestRTPDepayTrailingSEIWithMarker(t *testing.T) {
	codec := &core.Codec{Name: core.CodecH265}
	var aus [][]byte
	depay := RTPDepay(codec, func(p *rtp.Packet) { aus = append(aus, append([]byte(nil), p.Payload...)) }) // copy: the depacketizer reuses its buffer

	nal := func(typ byte, size int) []byte {
		b := make([]byte, size)
		b[0] = typ << 1
		b[1] = 1
		return b
	}
	seq := uint16(100)
	send := func(payload []byte, marker bool) {
		depay(&rtp.Packet{Header: rtp.Header{SequenceNumber: seq, Marker: marker}, Payload: payload})
		seq++
	}

	// AU 1: VPS, SPS, PPS, SEI, IDR (single NALs), then the next frame's SEI with the marker
	send(nal(NALUTypeVPS, 20), false)
	send(nal(NALUTypeSPS, 40), false)
	send(nal(NALUTypePPS, 8), false)
	send(nal(NALUTypePrefixSEI, 82), false)
	send(nal(NALUTypeIFrame, 900), false)
	send(nal(NALUTypePrefixSEI, 82), true)
	// AU 2: P slice, then trailing SEI with the marker
	send(nal(NALUTypePFrame, 300), false)
	send(nal(NALUTypePrefixSEI, 82), true)

	if len(aus) != 2 {
		t.Fatalf("expected 2 access units, got %d", len(aus))
	}
	if !IsKeyframe(aus[0]) {
		t.Fatalf("first AU should be a keyframe: types=%v len=%d", Types(aus[0]), len(aus[0]))
	}
	if IsKeyframe(aus[1]) {
		t.Fatalf("second AU should not be a keyframe")
	}
	// the trailing SEI must not be appended to the AU
	last := aus[1]
	size := int(binary.BigEndian.Uint32(last))
	if size+4 != len(last) || NALUType(last) != NALUTypePFrame {
		t.Fatalf("second AU should contain only the P slice, got len=%d", len(last))
	}
}
