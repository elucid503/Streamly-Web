package main

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

type ccMode string

const (
	ccUnknown ccMode = "unknown"
	ccNone ccMode = "none"
	ccRollup ccMode = "rollup"
	ccPopon ccMode = "popon"
	ccPainton ccMode = "painton"
	ccMixed ccMode = "mixed"
)

type ccSample struct {

	Mode ccMode
	Rollup int
	Popon int
	Painton int
	TextChars int
	Pairs int
	GA94 int
	Text string

}

func extractCC608(data []byte) ccSample {

	var s ccSample

	for i := 0; i < len(data); {

		j := bytes.Index(data[i:], []byte("GA94"))

		if j < 0 {

			break

		}

		i += j + 4
		s.GA94++

		if i >= len(data) || data[i] != 0x03 {

			continue

		}

		// user_data_type 0x03: process flags + cc_count in next byte
		if i+2 >= len(data) {

			break

		}

		i++
		flags := data[i]
		i++
		count := int(flags & 0x1F)

		if flags&0x40 == 0 {

			continue

		}

		// skip em_data
		if i >= len(data) {

			break

		}

		i++

		for n := 0; n < count && i+2 < len(data); n++ {

			marker := data[i]
			b1 := data[i+1]
			b2 := data[i+2]
			i += 3

			valid := marker&0x04 != 0
			ccType := marker & 0x03

			if !valid || ccType > 1 {

				continue

			}

			s.Pairs++
			classify608(&s, b1&0x7F, b2&0x7F)

		}

	}

	s.Mode = modeFromCounts(s)
	s.Text = strings.TrimSpace(s.Text)

	s.Text = collapseSpaces(s.Text)

	if len(s.Text) > 120 {

		s.Text = s.Text[:117] + "..."

	}

	return s

}

func classify608(s *ccSample, b1, b2 byte) {

	if b1 == 0 && b2 == 0 {

		return

	}

	// 0x10-0x1F: PAC, mid-row, and misc control — not letters.
	if b1 >= 0x10 && b1 <= 0x1F {

		if b1 == 0x14 || b1 == 0x15 || b1 == 0x1C || b1 == 0x1D {

			switch b2 {

			case 0x25, 0x26, 0x27:
				s.Rollup++
			case 0x20, 0x2F:
				s.Popon++
			case 0x29:
				s.Painton++
			}

		}

		// Row change / PAC: keep words from running together.
		if len(s.Text) > 0 && !strings.HasSuffix(s.Text, " ") {

			s.Text += " "

		}

		return

	}

	append608Text(s, b1)
	append608Text(s, b2)

}

func append608Text(s *ccSample, b byte) {

	if b < 0x20 || b == 0x7F {

		return

	}

	r := rune(b)

	if r == 0x2A {

		r = '\''

	}

	if unicode.IsPrint(r) {

		s.Text += string(r)
		s.TextChars++

	}

}

func modeFromCounts(s ccSample) ccMode {

	if s.GA94 == 0 {

		return ccUnknown

	}

	kinds := 0

	if s.Rollup > 0 {

		kinds++

	}

	if s.Popon > 0 {

		kinds++

	}

	if s.Painton > 0 {

		kinds++

	}

	if kinds == 0 {

		if s.TextChars == 0 && s.Pairs > 0 {

			return ccNone

		}

		if s.TextChars == 0 {

			return ccNone

		}

		return ccUnknown

	}

	if kinds > 1 {

		return ccMixed

	}

	switch {

	case s.Rollup > 0:
		return ccRollup
	case s.Popon > 0:
		return ccPopon
	default:
		return ccPainton

	}

}

func (s ccSample) AdLikely() bool {

	// GDELT: pop-on / paint-on / uncaptioned ≈ advertising on US news.
	switch s.Mode {

	case ccPopon, ccPainton, ccNone:
		return true

	default:
		return false

	}

}

func collapseSpaces(s string) string {

	return strings.Join(strings.Fields(s), " ")

}

func (s ccSample) String() string {

	return fmt.Sprintf("mode=%-7s ru=%d pop=%d paint=%d pairs=%d ga94=%d text=%q",
		s.Mode, s.Rollup, s.Popon, s.Painton, s.Pairs, s.GA94, s.Text)

}
