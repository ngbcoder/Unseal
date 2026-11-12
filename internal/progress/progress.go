// Package progress carries progress reporting through Unseal.
//
// A long operation reports a continuous ratio, but the interface needs
// increments: how much has it moved since the last update. A one-hour
// audiobook reporting every 16 KiB would flood the interface if every
// update carried an absolute position.
//
// The Reporter converts a ratio into increments. It remembers the last
// reported step and emits only the difference, and it emits the remainder
// when it is closed, so a bar always reaches the end.
//
// Layer L0 (foundation). It knows nothing about the operation it reports.
package progress

import "sync"

// Message is one progress update.
//
// A zero field means "not set". An increment is never zero, so the
// distinction between unset and zero does not matter in practice.
type Message struct {
	// ItemCount is the total number of items in the batch.
	ItemCount int
	// IncItem says that a new item has started, which resets the
	// per-item bar.
	IncItem int
	// IncStepsPerCent is the advance of the current item in per cent.
	IncStepsPerCent int
	// IncStepsPerMille is the advance of the current item in per mille.
	IncStepsPerMille int
}

// The maxima of the two granularities.
const (
	MaxPerMille = 1000
	MaxPerCent  = 100
)

// Reporter turns a continuous ratio into increments.
//
// A Reporter is safe for concurrent use.
type Reporter struct {
	mu sync.Mutex

	report      func(Message)
	max         int
	perMille    bool
	accumulated int
	closed      bool
}

// NewPerMille creates a reporter that counts in per mille, which is the
// granularity of the download and export bars.
func NewPerMille(report func(Message)) *Reporter {
	return &Reporter{report: report, max: MaxPerMille, perMille: true}
}

// NewPerCent creates a reporter that counts in per cent, which is the
// granularity of the decrypt bar.
func NewPerCent(report func(Message)) *Reporter {
	return &Reporter{report: report, max: MaxPerCent}
}

// Report records progress as a ratio between 0 and 1.
//
// The value is clamped, so a ratio above 1 does not overshoot. Only the
// difference since the last report is emitted, and nothing is emitted when
// the ratio has not moved far enough to change the step.
func (r *Reporter) Report(ratio float64) {
	if r == nil {
		return
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}

	total := int(ratio * float64(r.max))
	if total < 0 {
		total = 0
	}
	if total > r.max {
		total = r.max
	}

	inc := total - r.accumulated
	r.accumulated = total
	r.mu.Unlock()

	if inc > 0 {
		r.emit(inc)
	}
}

// Close emits the remainder, so that the bar reaches the end.
//
// It is safe to call more than once; only the first call emits anything.
func (r *Reporter) Close() {
	if r == nil {
		return
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true

	inc := r.max - r.accumulated
	r.accumulated = r.max
	r.mu.Unlock()

	if inc > 0 {
		r.emit(inc)
	}
}

// Accumulated returns the step reached so far.
func (r *Reporter) Accumulated() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.accumulated
}

// emit sends one increment.
func (r *Reporter) emit(inc int) {
	if r.report == nil {
		return
	}
	if r.perMille {
		r.report(Message{IncStepsPerMille: inc})
		return
	}
	r.report(Message{IncStepsPerCent: inc})
}
