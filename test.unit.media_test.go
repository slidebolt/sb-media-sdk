package media

import "testing"

func TestValidStream(t *testing.T) {
	valid := []string{"plugin-asterisk.ext_3000.audio", "a"}
	invalid := []string{"", ".", "a..b", "a.*", "a.>", "../x", "a b", "a/b", "a?b"}
	for _, s := range valid {
		if !ValidStream(s) {
			t.Errorf("%q should be valid", s)
		}
	}
	for _, s := range invalid {
		if ValidStream(s) {
			t.Errorf("%q should be invalid", s)
		}
	}
}

func TestStatusSubject(t *testing.T) {
	if got := StatusSubject("p.d.e"); got != "media.streams.p.d.e.status" {
		t.Fatalf("got %q", got)
	}
}
