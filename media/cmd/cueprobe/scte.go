package main

import (
	"encoding/binary"
	"fmt"
)

type scteMsg struct {

	Command string
	OutOfNetwork bool
	Cancel bool
	HasDuration bool
	DurationSec float64
	EventID uint32
	RawLen int

}

func parseSCTEMessages(data []byte) []scteMsg {

	if !looksLikeTS(data) {

		return nil

	}

	var out []scteMsg

	for _, pid := range tsPIDsOfType(data, 0x86) {

		for _, section := range tsAllSections(data, pid) {

			if msg, ok := parseSCTESection(section); ok {

				out = append(out, msg)

			}

		}

	}

	return out

}

func tsPIDsOfType(data []byte, streamType byte) []uint16 {

	pmtPID := tsPMTPID(data)

	if pmtPID == 0 {

		return nil

	}

	section := tsSection(data, pmtPID)

	if len(section) < 12 || section[0] != 0x02 {

		return nil

	}

	progInfoLen := int(binary.BigEndian.Uint16(section[10:12]) & 0x0FFF)
	i := 12 + progInfoLen

	var pids []uint16

	for i+5 <= len(section) {

		st := section[i]
		pid := binary.BigEndian.Uint16(section[i+1:i+3]) & 0x1FFF
		esLen := int(binary.BigEndian.Uint16(section[i+3:i+5]) & 0x0FFF)

		if st == streamType {

			pids = append(pids, pid)

		}

		i += 5 + esLen

	}

	return pids

}

func tsAllSections(data []byte, pid uint16) [][]byte {

	var (
		payload []byte
		out [][]byte
	)

	flush := func() {

		if len(payload) < 3 {

			payload = nil
			return

		}

		secLen := int(binary.BigEndian.Uint16(payload[1:3]) & 0x0FFF)

		if secLen < 1 || len(payload) < 3+secLen {

			payload = nil
			return

		}

		cp := make([]byte, 3+secLen)
		copy(cp, payload[:3+secLen])
		out = append(out, cp)
		payload = nil

	}

	for i := 0; i+188 <= len(data); i += 188 {

		if data[i] != 0x47 {

			continue

		}

		p := binary.BigEndian.Uint16(data[i+1:i+3]) & 0x1FFF

		if p != pid {

			continue

		}

		afc := (data[i+3] >> 4) & 0x3
		off := 4

		if afc == 2 || afc == 3 {

			aflen := int(data[i+4])
			off += 1 + aflen

			if off >= 188 {

				continue

			}

		}

		if afc != 1 && afc != 3 {

			continue

		}

		chunk := data[i+off : i+188]
		pusi := data[i+1]&0x40 != 0

		if pusi {

			flush()

			if len(chunk) < 1 {

				continue

			}

			ptr := int(chunk[0])
			chunk = chunk[1:]

			if ptr > 0 {

				if ptr > len(chunk) {

					continue

				}

				chunk = chunk[ptr:]

			}

			payload = append([]byte{}, chunk...)

		} else if len(payload) > 0 {

			payload = append(payload, chunk...)

		}

		if len(payload) >= 3 {

			secLen := int(binary.BigEndian.Uint16(payload[1:3]) & 0x0FFF)

			if len(payload) >= 3+secLen {

				flush()

			}

		}

	}

	flush()

	return out

}

func parseSCTESection(section []byte) (scteMsg, bool) {

	var msg scteMsg

	if len(section) < 14 || section[0] != 0xFC {

		return msg, false

	}

	// section_syntax_indicator and private_indicator must be 0.
	if section[1]&0xC0 != 0 {

		return msg, false

	}

	secLen := int(binary.BigEndian.Uint16(section[1:3]) & 0x0FFF)

	if secLen < 11 || len(section) < 3+secLen {

		return msg, false

	}

	msg.RawLen = 3 + secLen

	// After table_id(1)+len(2)+protocol(1)+enc/pts(5)+cw(1)+tier/cmdlen(3) = 13 bytes before command type
	if len(section) < 14 {

		return msg, false

	}

	cmdType := section[13]
	payload := section[14:]

	switch cmdType {

	case 0x00:
		msg.Command = "splice_null"
	case 0x04:
		msg.Command = "splice_schedule"
	case 0x05:
		msg.Command = "splice_insert"
		parseSpliceInsert(&msg, payload)
	case 0x06:
		msg.Command = "time_signal"
	case 0x07:
		msg.Command = "bandwidth_reservation"
	case 0xFF:
		msg.Command = "private_command"
	default:
		msg.Command = fmt.Sprintf("cmd_0x%02x", cmdType)

	}

	return msg, true

}

func parseSpliceInsert(msg *scteMsg, p []byte) {

	if len(p) < 5 {

		return

	}

	msg.EventID = binary.BigEndian.Uint32(p[0:4])
	msg.Cancel = p[4]&0x80 != 0

	if msg.Cancel || len(p) < 6 {

		return

	}

	flags := p[5]
	msg.OutOfNetwork = flags&0x80 != 0
	durationFlag := flags&0x20 != 0
	immediate := flags&0x10 != 0
	programSplice := flags&0x40 != 0

	i := 6

	if programSplice && !immediate {

		// splice_time()
		if i >= len(p) {

			return

		}

		if p[i]&0x80 != 0 {

			i += 5

		} else {

			i++

		}

	}

	if durationFlag && i+5 <= len(p) {

		// break_duration: auto_return(1) + reserved(6) + duration(33) as 90kHz
		raw := uint64(p[i]&0x01)<<32 | uint64(binary.BigEndian.Uint32(p[i+1:i+5]))
		msg.HasDuration = true
		msg.DurationSec = float64(raw) / 90000.0

	}

}

func summarizeSCTE(msgs []scteMsg) string {

	if len(msgs) == 0 {

		return "scte-pid-empty"

	}

	out := false
	in := false
	cmds := map[string]int{}

	for _, m := range msgs {

		cmds[m.Command]++

		if m.Command == "splice_insert" && !m.Cancel {

			if m.OutOfNetwork {

				out = true

			} else {

				in = true

			}

		}

	}

	switch {

	case out && !in:
		return fmt.Sprintf("OUT (%d msgs)", len(msgs))
	case in && !out:
		return fmt.Sprintf("IN (%d msgs)", len(msgs))
	case out && in:
		return fmt.Sprintf("OUT+IN (%d msgs)", len(msgs))
	default:
		return fmt.Sprintf("%v x%d", cmds, len(msgs))

	}

}
