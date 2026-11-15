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

// satsText carries internal/web's sats table row for row, so the two copies of
// §9's rule read an amount the same way — the negative sub-sat rows included,
// since msat/1000 alone loses the sign of -500.
func TestSatsTextMatchesTheWebFormattersTable(t *testing.T) {
	for _, c := range []struct {
		msat int64
		want string
	}{
		{0, "0 sats"},
		{1_000, "1 sats"},
		{1_500, "1.5 sats"},
		{1_001, "1.001 sats"},
		{21_000_000_000, "21000000 sats"},
		{-1_500, "-1.5 sats"},
		{-500, "-0.5 sats"},
		{1_010, "1.01 sats"},
		{10, "0.01 sats"},
		{-1, "-0.001 sats"},
		{math.MinInt64, "-9223372036854775.808 sats"},
	} {
		if got := satsText(c.msat); got != c.want {
			t.Errorf("satsText(%d) = %q, want %q", c.msat, got, c.want)
		}
	}
}

// A CAP of zero says what it does, because the operator's sentence is where they
// learn it; any other cap reads exactly as satsText does.
func TestMsatSentenceWarnsOnlyForAZeroCap(t *testing.T) {
	if got, want := msatSentence(0), "0 sats (which refuses every payment)"; got != want {
		t.Errorf("msatSentence(0) = %q, want %q", got, want)
	}
	if got, want := msatSentence(1_500), satsText(1_500); got != want {
		t.Errorf("msatSentence(1500) = %q, want satsText's %q", got, want)
	}
}
