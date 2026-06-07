package pkg

import (
	"strconv"
	"strings"
)

// lossless textual encoding of a BlockModel for konfi's string editor channel.
//
// format: a flat sequence of length-prefixed records. each record is
//
//	<tag>\x20<byteLen>\n<exactly byteLen bytes>\n
//
// the trailing \n after the payload is a fixed separator, not part of the
// payload, so payloads may contain arbitrary bytes (newlines, quotes, pipes,
// the encoding's own tag words) without ambiguity. records appear in a fixed
// order with no maps anywhere, so Encode is deterministic and Decode∘Encode is
// the exact identity.
//
// record stream:
//
//	BM <n>            -> n blocks follow
//	  B <id>          -> block id
//	  OP <opener>
//	  HD <header>
//	  SP <m>          -> m RawSpan lines follow
//	    SL <line>     -> one raw span line (with terminator) ×m
//	  BD <k>          -> k body entries follow
//	    E <id>        -> entry id
//	    EK <kind>
//	    EY <key>
//	    EV <p>        -> p value tokens follow
//	      VL <val>    -> one value ×p
//	    ER <q>        -> q raw lines follow
//	      RL <line>   -> one raw line ×q

const blkVersion = "BMv1"

// Encode serializes m to a deterministic, lossless string.
func Encode(m BlockModel) string {
	var sb strings.Builder
	writeRec(&sb, "V", blkVersion)
	writeRec(&sb, "BM", strconv.Itoa(len(m.Blocks)))
	for _, b := range m.Blocks {
		writeRec(&sb, "B", b.ID)
		writeRec(&sb, "OP", b.Opener)
		writeRec(&sb, "HD", b.Header)
		writeRec(&sb, "SP", strconv.Itoa(len(b.RawSpan)))
		for _, l := range b.RawSpan {
			writeRec(&sb, "SL", l)
		}
		writeRec(&sb, "BD", strconv.Itoa(len(b.Body)))
		for _, e := range b.Body {
			writeRec(&sb, "E", e.ID)
			writeRec(&sb, "EK", e.Kind)
			writeRec(&sb, "EY", e.Key)
			writeRec(&sb, "EV", strconv.Itoa(len(e.Values)))
			for _, v := range e.Values {
				writeRec(&sb, "VL", v)
			}
			writeRec(&sb, "ER", strconv.Itoa(len(e.Raw)))
			for _, r := range e.Raw {
				writeRec(&sb, "RL", r)
			}
		}
	}
	return sb.String()
}

func writeRec(sb *strings.Builder, tag, payload string) {
	sb.WriteString(tag)
	sb.WriteByte(' ')
	sb.WriteString(strconv.Itoa(len(payload)))
	sb.WriteByte('\n')
	sb.WriteString(payload)
	sb.WriteByte('\n')
}

// decoder is a small cursor over an Encode stream.
type decoder struct {
	s   string
	pos int
}

// next reads one record, returning its tag and payload. ok is false at EOF.
func (d *decoder) next() (tag, payload string, ok bool) {
	if d.pos >= len(d.s) {
		return "", "", false
	}
	nl := strings.IndexByte(d.s[d.pos:], '\n')
	if nl < 0 {
		return "", "", false
	}
	head := d.s[d.pos : d.pos+nl]
	d.pos += nl + 1
	sp := strings.LastIndexByte(head, ' ')
	if sp < 0 {
		return "", "", false
	}
	tag = head[:sp]
	n, err := strconv.Atoi(head[sp+1:])
	if err != nil || d.pos+n > len(d.s) {
		return "", "", false
	}
	payload = d.s[d.pos : d.pos+n]
	d.pos += n
	// consume the fixed payload-terminating newline
	if d.pos < len(d.s) && d.s[d.pos] == '\n' {
		d.pos++
	}
	return tag, payload, true
}

// Decode parses an Encode stream back into a BlockModel. it is the exact inverse
// of Encode for any well-formed stream; malformed input yields a best-effort
// partial model.
func Decode(s string) BlockModel {
	d := &decoder{s: s}
	var m BlockModel

	// version record (ignored beyond presence)
	if tag, _, ok := d.next(); !ok || tag != "V" {
		return m
	}
	tag, payload, ok := d.next()
	if !ok || tag != "BM" {
		return m
	}
	nBlocks, _ := strconv.Atoi(payload)
	if nBlocks > 0 {
		m.Blocks = make([]Block, 0, nBlocks)
	}

	for bi := 0; bi < nBlocks; bi++ {
		var b Block
		if p, ok := expect(d, "B"); ok {
			b.ID = p
		}
		if p, ok := expect(d, "OP"); ok {
			b.Opener = p
		}
		if p, ok := expect(d, "HD"); ok {
			b.Header = p
		}
		if p, ok := expect(d, "SP"); ok {
			n, _ := strconv.Atoi(p)
			if n > 0 {
				b.RawSpan = make([]string, 0, n)
			}
			for k := 0; k < n; k++ {
				if l, ok := expect(d, "SL"); ok {
					b.RawSpan = append(b.RawSpan, l)
				}
			}
		}
		if p, ok := expect(d, "BD"); ok {
			n, _ := strconv.Atoi(p)
			if n > 0 {
				b.Body = make([]Entry, 0, n)
			}
			for k := 0; k < n; k++ {
				b.Body = append(b.Body, decodeEntry(d))
			}
		}
		m.Blocks = append(m.Blocks, b)
	}
	return m
}

func decodeEntry(d *decoder) Entry {
	var e Entry
	if p, ok := expect(d, "E"); ok {
		e.ID = p
	}
	if p, ok := expect(d, "EK"); ok {
		e.Kind = p
	}
	if p, ok := expect(d, "EY"); ok {
		e.Key = p
	}
	if p, ok := expect(d, "EV"); ok {
		n, _ := strconv.Atoi(p)
		if n > 0 {
			e.Values = make([]string, 0, n)
		}
		for k := 0; k < n; k++ {
			if v, ok := expect(d, "VL"); ok {
				e.Values = append(e.Values, v)
			}
		}
	}
	if p, ok := expect(d, "ER"); ok {
		n, _ := strconv.Atoi(p)
		if n > 0 {
			e.Raw = make([]string, 0, n)
		}
		for k := 0; k < n; k++ {
			if r, ok := expect(d, "RL"); ok {
				e.Raw = append(e.Raw, r)
			}
		}
	}
	return e
}

// expect reads the next record and verifies its tag, returning the payload.
func expect(d *decoder, want string) (payload string, ok bool) {
	tag, payload, ok := d.next()
	if !ok || tag != want {
		return "", false
	}
	return payload, true
}
