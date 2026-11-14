package guard

import (
	"math"
	"testing"
)

// The operator's sentences, written out, because nothing else pins them.
//
// The agreement tests above check that the trail and the file SAY THE SAME
// THING; neither notices if both are reworded together. These are the words the
// operator reads in the authorisation file and compares against what they asked
// for, and that comparison is the whole reason typing a code is safe — so the
// words themselves are a security control, and a golden is the only thing that
// makes changing them deliberate.
//
// An internal test because describe() and sentence() are unexported: the
// alternative is to reach them through a real ceremony, which is what the tests
// above already do and which cannot pin an exact string cheaply.
func TestTheOperatorFacingSentencesAreExactlyThese(t *testing.T) {
	for _, c := range []struct {
		change  Change
		file    string
		lowered string
	}{
		{Change{Control: ControlSending, On: true},
			"TURN SENDING ON — let this app make your node pay invoices.",
			"TURN SENDING ON — let this app make your node pay invoices."},
		{Change{Control: ControlSending},
			"turn sending off.", "turn sending off."},
		{Change{Control: ControlSpendCap, Msat: 50_000_000},
			"RAISE THE SPENDING LIMIT to 50000 sats in any 24 hours.",
			"LOWER THE SPENDING LIMIT to 50000 sats in any 24 hours."},
		{Change{Control: ControlPaymentCap, Msat: 1_000_000},
			"RAISE THE PER-PAYMENT LIMIT to 1000 sats.",
			"LOWER THE PER-PAYMENT LIMIT to 1000 sats."},
		{Change{Control: ControlSpendCap, Msat: 50_000_500},
			"RAISE THE SPENDING LIMIT to 50000.5 sats in any 24 hours.",
			"LOWER THE SPENDING LIMIT to 50000.5 sats in any 24 hours."},
	} {
		if got := c.change.describe(); got != c.file {
			t.Errorf("the authorisation file would say\n  %q\nwant\n  %q\nThis is the sentence "+
				"the operator checks against what they asked for; changing it is a decision, "+
				"not a tidy-up.", got, c.file)
		}
		// A raise renders identically in the trail, which is what lets an
		// operator read a row against the file they kept.
		if got := c.change.auditSentence(false); got != c.file {
			t.Errorf("the trail says %q for a raise and the file says %q", got, c.file)
		}
		if got := c.change.auditSentence(true); got != c.lowered {
			t.Errorf("the trail says\n  %q\nfor a lowering, want\n  %q", got, c.lowered)
		}
	}
}

// The edge cases of internal/web's sats table, so the two copies of §9's rule
// read an amount the same way; zero keeps its warning, because the sentence is
// where the operator learns that a zero cap refuses everything.
func TestMsatSentenceRendersSatsWithTheRemainderOnlyWhenNonZero(t *testing.T) {
	for _, c := range []struct {
		msat int64
		want string
	}{
		{0, "0 sats (which refuses every payment)"},
		{1_000, "1 sats"},
		{1_500, "1.5 sats"},
		{1_001, "1.001 sats"},
		{1_010, "1.01 sats"},
		{500, "0.5 sats"},
		{1, "0.001 sats"},
		{21_000_000_000, "21000000 sats"},
		{-1_500, "-1.5 sats"}, // valid() refuses one first; the sign must not garble
		{math.MinInt64, "-9223372036854775.808 sats"},
	} {
		if got := msatSentence(c.msat); got != c.want {
			t.Errorf("msatSentence(%d) = %q, want %q", c.msat, got, c.want)
		}
	}
}
