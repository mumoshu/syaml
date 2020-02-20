package syaml

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestMatch(t *testing.T) {
	data := []byte(`a: "1"

b:
  bb: "2"

c:
  cc:
    ccc: "3"
`)

	doc := yaml.Node{}

	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("%v", err)
	}

	type testcase struct {
		cond *Condition
		want bool
	}

	testcases := []testcase{
		{
			cond: Eq([]string{"a"}, "1"),
			want: true,
		},
		{
			cond: Eq([]string{"a"}, "2"),
			want: false,
		},
		{
				cond: And(Eq([]string{"a"}, "1"), Eq([]string{"b", "bb"}, "2")),
				want: true,
		},
		{
				cond: And(Eq([]string{"a"}, "1"), Eq([]string{"b", "bb"}, "3")),
				want: false,
		},
		{
				cond: And(Eq([]string{"a"}, "2"), Eq([]string{"b", "bb"}, "2")),
				want: false,
		},
		{
			cond: And(Eq([]string{"a"}, "2"), Eq([]string{"b", "bb"}, "3")),
			want: false,
		},
		{
				cond: Or(Eq([]string{"a"}, "1"), Eq([]string{"b", "bb"}, "2")),
				want: true,
		},
		{
				cond: Or(Eq([]string{"a"}, "1"), Eq([]string{"b", "bb"}, "3")),
				want: true,
		},
		{
				cond: Or(Eq([]string{"a"}, "2"), Eq([]string{"b", "bb"}, "2")),
				want: true,
		},
		{
				cond: Or(Eq([]string{"a"}, "2"), Eq([]string{"b", "bb"}, "3")),
				want: false,
		},
	}

	for i := range testcases {
		tc := testcases[i]

		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got, err := Match(&doc, tc.cond)
			if err != nil {
				t.Errorf("%v", err)
			}

			if got != tc.want {
				t.Errorf("unexpcted result: want %v, got %v", tc.want, got)
			}
		})
	}
}
