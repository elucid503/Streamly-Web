package sports

import "testing"

func TestEspnIsDelayed(t *testing.T) {

	cases := []struct {

		name string
		short string
		detail string
		desc string
		want bool

	}{

		{name: "STATUS_RAIN_DELAY", want: true},
		{name: "STATUS_DELAYED", detail: "Delayed", want: true},
		{name: "STATUS_SUSPENDED", want: true},
		{name: "STATUS_IN_PROGRESS", short: "Rain Delay", want: true},
		{name: "STATUS_SCHEDULED", short: "7:05 PM", want: false},
		{name: "STATUS_IN_PROGRESS", short: "Top 3rd", want: false},

	}

	for _, tc := range cases {

		got := espnIsDelayed(tc.name, tc.short, tc.detail, tc.desc)

		if got != tc.want {

			t.Fatalf("%s / %s: got %v want %v", tc.name, tc.short, got, tc.want)

		}

	}

}
