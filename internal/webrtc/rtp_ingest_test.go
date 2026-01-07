package webrtc

import (
	"testing"

	"github.com/pion/rtp"
)

// TestRTPRewriterSequence ensures outgoing sequence numbers are contiguous regardless of input sequence.
func TestRTPRewriterSequence(t *testing.T) {
	var rw rtpRewriter
	p := &rtp.Packet{Header: rtp.Header{SequenceNumber: 100, Timestamp: 10, PayloadType: 96, SSRC: 1}}
	rw.Apply(p, rtpWriteParams{})
	first := p.SequenceNumber

	p2 := &rtp.Packet{Header: rtp.Header{SequenceNumber: 1, Timestamp: 20, PayloadType: 96, SSRC: 1}}
	rw.Apply(p2, rtpWriteParams{})
	if p2.SequenceNumber != first+1 {
		t.Fatalf("expected contiguous sequence, got %d then %d", first, p2.SequenceNumber)
	}
}

// TestRTPRewriterTimestampGrouping keeps all packets with the same input timestamp on the same output timestamp.
func TestRTPRewriterTimestampGrouping(t *testing.T) {
	var rw rtpRewriter

	p1 := &rtp.Packet{Header: rtp.Header{SequenceNumber: 1, Timestamp: 1000}}
	rw.Apply(p1, rtpWriteParams{})
	base := p1.Timestamp

	p2 := &rtp.Packet{Header: rtp.Header{SequenceNumber: 2, Timestamp: 1000}}
	rw.Apply(p2, rtpWriteParams{})
	if p2.Timestamp != base {
		t.Fatalf("expected same output timestamp for same input timestamp, got %d != %d", p2.Timestamp, base)
	}

	p3 := &rtp.Packet{Header: rtp.Header{SequenceNumber: 3, Timestamp: 1300}}
	rw.Apply(p3, rtpWriteParams{})
	if p3.Timestamp <= base {
		t.Fatalf("expected output timestamp to advance, got %d <= %d", p3.Timestamp, base)
	}
}

// TestRTPRewriterLargeJump avoids forwarding a huge delta on discontinuities and stays monotonic.
func TestRTPRewriterLargeJump(t *testing.T) {
	var rw rtpRewriter

	p1 := &rtp.Packet{Header: rtp.Header{SequenceNumber: 1, Timestamp: 5000}}
	rw.Apply(p1, rtpWriteParams{})
	t1 := p1.Timestamp

	p2 := &rtp.Packet{Header: rtp.Header{SequenceNumber: 2, Timestamp: 8000}}
	rw.Apply(p2, rtpWriteParams{})
	t2 := p2.Timestamp
	if t2 <= t1 {
		t.Fatalf("expected monotonic timestamp, got %d <= %d", t2, t1)
	}

	// Simulate ffmpeg restart: timestamp jumps far backwards, which would produce a huge unsigned delta.
	p3 := &rtp.Packet{Header: rtp.Header{SequenceNumber: 1, Timestamp: 10}}
	rw.Apply(p3, rtpWriteParams{})
	t3 := p3.Timestamp
	if t3 <= t2 {
		t.Fatalf("expected monotonic timestamp after jump, got %d <= %d", t3, t2)
	}

	// Another packet with same input TS should keep the same output TS.
	p4 := &rtp.Packet{Header: rtp.Header{SequenceNumber: 2, Timestamp: 10}}
	rw.Apply(p4, rtpWriteParams{})
	if p4.Timestamp != t3 {
		t.Fatalf("expected grouped timestamp after restart, got %d != %d", p4.Timestamp, t3)
	}
}

// TestRTPRewriterOverridesHeader verifies payload type and SSRC overrides are applied when set.
func TestRTPRewriterOverridesHeader(t *testing.T) {
	var rw rtpRewriter
	p := &rtp.Packet{Header: rtp.Header{SequenceNumber: 1, Timestamp: 1, PayloadType: 96, SSRC: 123}}
	rw.Apply(p, rtpWriteParams{payloadType: 120})
	if p.PayloadType != 120 {
		t.Fatalf("expected payload type override, got %d", p.PayloadType)
	}
}
