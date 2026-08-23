package main

import (
	"bytes"
	"encoding/binary"
)

func scanSegment(data []byte) []string {

	var hits []string
	seen := map[string]bool{}

	add := func(label string) {

		if !seen[label] {

			seen[label] = true
			hits = append(hits, label)

		}

	}

	if looksLikeTS(data) {

		types := tsStreamTypes(data)
		hasSCTEPID := false

		for _, t := range types {

			if t == 0x86 {

				hasSCTEPID = true
				add("PMT stream_type=0x86 (SCTE-35)")

			}

		}

		// Only treat 0xFC as SCTE-35 when the PMT actually declared that PID.
		// Bare 0xFC matches are common in video PES and are not cues.
		if hasSCTEPID && tsHasTableID(data, 0xFC) {

			add("TS table_id=0xFC (SCTE-35 section)")

		}

		if bytes.Contains(data, []byte("GA94")) {

			add("GA94 (CEA-608/708 in video user data)")

		}

	}

	if bytes.Contains(data, []byte("emsg")) {

		add("fMP4 emsg box")

	}

	return hits

}

func looksLikeTS(data []byte) bool {

	if len(data) < 188 {

		return false

	}

	hits := 0

	for i := 0; i+188 <= len(data) && i < 188*8; i += 188 {

		if data[i] == 0x47 {

			hits++

		}

	}

	return hits >= 3

}

func tsStreamTypes(data []byte) []byte {

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

	var types []byte

	for i+5 <= len(section) {

		st := section[i]
		esLen := int(binary.BigEndian.Uint16(section[i+3:i+5]) & 0x0FFF)
		types = append(types, st)
		i += 5 + esLen

	}

	return types

}

func tsPMTPID(data []byte) uint16 {

	section := tsSection(data, 0)

	if len(section) < 16 || section[0] != 0x00 {

		return 0

	}

	i := 8

	for i+4 <= len(section)-4 {

		prog := binary.BigEndian.Uint16(section[i : i+2])
		pid := binary.BigEndian.Uint16(section[i+2:i+4]) & 0x1FFF
		i += 4

		if prog != 0 && pid != 0x1FFF {

			return pid

		}

	}

	return 0

}

func tsSection(data []byte, pid uint16) []byte {

	var payload []byte

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

			payload = append(payload[:0], chunk...)

		} else if len(payload) > 0 {

			payload = append(payload, chunk...)

		}

		if len(payload) >= 3 {

			secLen := int(binary.BigEndian.Uint16(payload[1:3]) & 0x0FFF)

			if len(payload) >= 3+secLen {

				return payload[:3+secLen]

			}

		}

	}

	return payload

}

func tsHasTableID(data []byte, tableID byte) bool {

	for i := 0; i+188 <= len(data); i += 188 {

		if data[i] != 0x47 {

			continue

		}

		afc := (data[i+3] >> 4) & 0x3
		off := 4

		if afc == 2 || afc == 3 {

			aflen := int(data[i+4])
			off += 1 + aflen

		}

		if off >= i+188 || off >= 188 {

			continue

		}

		if bytes.IndexByte(data[i+off:i+188], tableID) >= 0 && tableID == 0xFC {

			// Cheap filter: SCTE-35 sections start with 0xFC and a 12-bit length.
			chunk := data[i+off : i+188]

			for j := 0; j+3 < len(chunk); j++ {

				if chunk[j] == 0xFC {

					l := int(binary.BigEndian.Uint16(chunk[j+1:j+3]) & 0x0FFF)

					if l >= 10 && l < 500 {

						return true

					}

				}

			}

		}

	}

	return false

}
