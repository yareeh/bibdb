package internal

import "testing"

func TestLaTeXToUTF8(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{`H{\"a}meenlinna`, `Hämeenlinna`},
		{`Katoava j{\"a}{\"a}`, `Katoava jää`},
		{`Fr{\'e}d{\'e}ric`, `Frédéric`},
		{`Mik{\"a} on datatuote?`, `Mikä on datatuote?`},
		{`M{\"u}ller`, `Müller`},
		{`na\"ive`, `naïve`},              // bare accent without braces
		{`caf{\'e}`, `café`},              // common word
		{`{\"o}`, `ö`},                    // standalone
		{`no accents here`, `no accents here`},
		{`Faber \& Faber`, `Faber & Faber`},
		{`50\% done`, `50% done`},
		{`{\"A}land`, `Äland`},            // uppercase
		{`G{\"{o}}del`, `Gödel`},          // letter in braces
	}
	for _, tt := range tests {
		got := LaTeXToUTF8(tt.in)
		if got != tt.want {
			t.Errorf("LaTeXToUTF8(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
