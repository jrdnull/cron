package cron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want Expression
		err  string
	}{
		{in: "", err: "invalid expression"},
		{in: "* * * * *", err: "invalid expression"},
		{
			in:  "a b c d e f /bin/test",
			err: "invalid value",
		},
		{in: "* * * * *", err: "invalid expression"},
		{
			in:  "-1 0 1 1 1 /bin/test",
			err: "invalid range",
		},
		{
			in:  "123 0 1 1 1 /bin/test",
			err: "outside of range: 0-59",
		},
		{
			in:  "1-123 0 1 1 1 /bin/test",
			err: "outside of range: 0-59",
		},
		{
			in:  "0 0 1 1-two 1 /bin/test",
			err: "invalid range end: two",
		},
		{
			in:  "0 0 1 1-1-1 1 /bin/test",
			err: "invalid range",
		},
		{
			in:  "0 0 1 */x 1 /bin/test",
			err: "invalid step range",
		},
		{
			in:  "0 0 1 1-5/x 1 /bin/test",
			err: "invalid step range",
		},
		{
			in:  "0 0 1 1-5/1/2 1 /bin/test",
			err: "invalid step range",
		},
		{
			in:  "0 0 1 1,2,5-100 1 /bin/test",
			err: "outside of range",
		},
		{
			in:  "5-2 0 1 1 1 /bin/test",
			err: "invalid range",
		},
		{ // step too big should include initial value
			in: "*/100 0 1 1 1 /bin/test",
			want: Expression{
				Minute:     []uint8{0},
				Hour:       []uint8{0},
				DayOfMonth: []uint8{1},
				Month:      []uint8{1},
				DayOfWeek:  []uint8{1},
				Command:    "/bin/test",
			},
		},
		{
			in: "0 0 1 jaN SuN /bin/names",
			want: Expression{
				Minute:     []uint8{0},
				Hour:       []uint8{0},
				DayOfMonth: []uint8{1},
				Month:      []uint8{1},
				DayOfWeek:  []uint8{7},
				Command:    "/bin/names",
			},
		},
		{
			in: `0 0 1 1 1 echo "hello, world!"`,
			want: Expression{
				Minute:     []uint8{0},
				Hour:       []uint8{0},
				DayOfMonth: []uint8{1},
				Month:      []uint8{1},
				DayOfWeek:  []uint8{1},
				Command:    `echo "hello, world!"`,
			},
		},
		{
			in: "0,1,2 0 1 1 1 /bin/test",
			want: Expression{
				Minute:     []uint8{0, 1, 2},
				Hour:       []uint8{0},
				DayOfMonth: []uint8{1},
				Month:      []uint8{1},
				DayOfWeek:  []uint8{1},
				Command:    "/bin/test",
			},
		},
		{
			in: "0,1,2,10-59/10 0 1 1 1 /bin/test",
			want: Expression{
				Minute:     []uint8{0, 1, 2, 10, 20, 30, 40, 50},
				Hour:       []uint8{0},
				DayOfMonth: []uint8{1},
				Month:      []uint8{1},
				DayOfWeek:  []uint8{1},
				Command:    "/bin/test",
			},
		},
		{
			in: "0-5,50-55 0 1 1 1 /bin/test",
			want: Expression{
				Minute: []uint8{
					0, 1, 2, 3, 4, 5, 50, 51, 52, 53, 54, 55,
				},
				Hour:       []uint8{0},
				DayOfMonth: []uint8{1},
				Month:      []uint8{1},
				DayOfWeek:  []uint8{1},
				Command:    "/bin/test",
			},
		},
		{
			in: "* * * * * /bin/test",
			want: Expression{
				Minute: []uint8{
					0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
					11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
					21, 22, 23, 24, 25, 26, 27, 28, 29, 30,
					31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
					41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
					51, 52, 53, 54, 55, 56, 57, 58, 59,
				},
				Hour: []uint8{
					0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
					11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
					21, 22, 23,
				},
				DayOfMonth: []uint8{
					1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
					11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
					21, 22, 23, 24, 25, 26, 27, 28, 29, 30,
					31,
				},
				Month: []uint8{
					1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
				},
				DayOfWeek: []uint8{1, 2, 3, 4, 5, 6, 7},
				Command:   "/bin/test",
			},
		},
		{
			in: "*/15 0 1,15 * 1-5 /usr/bin/find",
			want: Expression{
				Minute:     []uint8{0, 15, 30, 45},
				Hour:       []uint8{0},
				DayOfMonth: []uint8{1, 15},
				Month: []uint8{
					1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
				},
				DayOfWeek: []uint8{1, 2, 3, 4, 5},
				Command:   "/usr/bin/find",
			},
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("Parse(%v)", tc.in), func(t *testing.T) {
			got, err := Parse(tc.in)
			if tc.err == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.err != "" {
				if err == nil {
					t.Fatal("expected error")
				} else if !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("err got: %v, want %v", err, tc.err)
				}
			}

			if diff := cmp.Diff(got.String(), tc.want.String()); diff != "" {
				t.Errorf("(-got +want)\n:%s", diff)
			}
		})
	}
}
