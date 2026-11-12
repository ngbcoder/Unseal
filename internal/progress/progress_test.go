package progress_test

import (
	"testing"

	"github.com/ngbcoder/Unseal/internal/progress"
)

func TestPerMilleReportsIncrements(t *testing.T) {
	var got []progress.Message

	r := progress.NewPerMille(func(m progress.Message) { got = append(got, m) })

	r.Report(0.5)

	if len(got) != 1 {
		t.Fatalf("expected one message, got %d", len(got))
	}
	if got[0].IncStepsPerMille != 500 {
		t.Errorf("the increment is %d, want 500", got[0].IncStepsPerMille)
	}
	if got[0].IncStepsPerCent != 0 {
		t.Errorf("a per-mille reporter must not set the per-cent increment")
	}
}

func TestPerCentReportsIncrements(t *testing.T) {
	var got []progress.Message

	r := progress.NewPerCent(func(m progress.Message) { got = append(got, m) })

	r.Report(0.25)

	if len(got) != 1 {
		t.Fatalf("expected one message, got %d", len(got))
	}
	if got[0].IncStepsPerCent != 25 {
		t.Errorf("the increment is %d, want 25", got[0].IncStepsPerCent)
	}
	if got[0].IncStepsPerMille != 0 {
		t.Errorf("a per-cent reporter must not set the per-mille increment")
	}
}

func TestOnlyTheDifferenceIsReported(t *testing.T) {
	var got []progress.Message

	r := progress.NewPerMille(func(m progress.Message) { got = append(got, m) })

	r.Report(0.100)
	r.Report(0.200)
	r.Report(0.250)

	if len(got) != 3 {
		t.Fatalf("expected three messages, got %d", len(got))
	}
	if got[0].IncStepsPerMille != 100 {
		t.Errorf("the first increment is %d, want 100", got[0].IncStepsPerMille)
	}
	if got[1].IncStepsPerMille != 100 {
		t.Errorf("the second increment is %d, want 100", got[1].IncStepsPerMille)
	}
	if got[2].IncStepsPerMille != 50 {
		t.Errorf("the third increment is %d, want 50", got[2].IncStepsPerMille)
	}
}

func TestTinyStepsAreNotReported(t *testing.T) {
	var got []progress.Message

	r := progress.NewPerMille(func(m progress.Message) { got = append(got, m) })

	// A ratio that does not move the step must not produce a message.
	r.Report(0.0001)
	r.Report(0.0002)

	if len(got) != 0 {
		t.Errorf("expected no messages, got %d", len(got))
	}
}

func TestTheRatioIsClamped(t *testing.T) {
	var got []progress.Message

	r := progress.NewPerMille(func(m progress.Message) { got = append(got, m) })

	r.Report(2.0)

	if len(got) != 1 {
		t.Fatalf("expected one message, got %d", len(got))
	}
	if got[0].IncStepsPerMille != progress.MaxPerMille {
		t.Errorf("the increment is %d, want %d", got[0].IncStepsPerMille, progress.MaxPerMille)
	}
}

func TestANegativeRatioIsClamped(t *testing.T) {
	var got []progress.Message

	r := progress.NewPerMille(func(m progress.Message) { got = append(got, m) })

	r.Report(-1.0)

	if len(got) != 0 {
		t.Errorf("expected no messages, got %d", len(got))
	}
}

func TestCloseEmitsTheRemainder(t *testing.T) {
	var got []progress.Message

	r := progress.NewPerMille(func(m progress.Message) { got = append(got, m) })

	r.Report(0.4)
	r.Close()

	if len(got) != 2 {
		t.Fatalf("expected two messages, got %d", len(got))
	}
	if got[1].IncStepsPerMille != 600 {
		t.Errorf("the remainder is %d, want 600", got[1].IncStepsPerMille)
	}
}

func TestTheIncrementsSumToTheMaximum(t *testing.T) {
	var total int

	r := progress.NewPerMille(func(m progress.Message) { total += m.IncStepsPerMille })

	for i := 0; i <= 100; i++ {
		r.Report(float64(i) / 100)
	}
	r.Close()

	if total != progress.MaxPerMille {
		t.Errorf("the increments sum to %d, want %d", total, progress.MaxPerMille)
	}
}

func TestTheIncrementsAreMonotonic(t *testing.T) {
	var total int

	r := progress.NewPerCent(func(m progress.Message) { total += m.IncStepsPerCent })

	for i := 0; i <= 100; i++ {
		r.Report(float64(i) / 100)
	}
	r.Close()

	if total != progress.MaxPerCent {
		t.Errorf("the increments sum to %d, want %d", total, progress.MaxPerCent)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	var got []progress.Message

	r := progress.NewPerMille(func(m progress.Message) { got = append(got, m) })

	r.Report(0.5)
	r.Close()
	r.Close()
	r.Close()

	if len(got) != 2 {
		t.Fatalf("expected two messages, got %d", len(got))
	}
}

func TestReportingAfterCloseIsIgnored(t *testing.T) {
	var got []progress.Message

	r := progress.NewPerMille(func(m progress.Message) { got = append(got, m) })

	r.Close()
	r.Report(0.5)

	if len(got) != 1 {
		t.Errorf("expected one message, got %d", len(got))
	}
}

func TestAccumulated(t *testing.T) {
	r := progress.NewPerMille(nil)

	r.Report(0.25)
	if r.Accumulated() != 250 {
		t.Errorf("the accumulated value is %d, want 250", r.Accumulated())
	}

	r.Close()
	if r.Accumulated() != progress.MaxPerMille {
		t.Errorf("the accumulated value is %d, want %d", r.Accumulated(), progress.MaxPerMille)
	}
}

func TestANilReporterIsHarmless(t *testing.T) {
	var r *progress.Reporter

	r.Report(0.5)
	r.Close()

	if r.Accumulated() != 0 {
		t.Error("a nil reporter must report nothing")
	}
}

func TestAReportWithoutACallbackIsHarmless(t *testing.T) {
	r := progress.NewPerMille(nil)

	r.Report(0.5)
	r.Close()
}
