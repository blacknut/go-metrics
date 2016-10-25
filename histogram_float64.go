package metrics

type HistogramFloat64 interface {
	Clear()
	Count() int64
	Max() float64
	Mean() float64
	Min() float64
	Percentile(float64) float64
	Percentiles([]float64) []float64
	Sample() SampleFloat64
	Snapshot() HistogramFloat64
	StdDev() float64
	Sum() float64
	Update(float64)
	Variance() float64
}

func NewHistogramFloat64(s SampleFloat64) HistogramFloat64 {
	return &StandardHistogramFloat64{sample: s}
}

// HistogramSnapshot is a read-only copy of another Histogram.
type HistogramSnapshotFloat64 struct {
	sample *SampleSnapshotFloat64
}

// Clear panics.
func (*HistogramSnapshotFloat64) Clear() {
	panic("Clear called on a HistogramSnapshotFloat64")
}

// Count returns the number of samples recorded at the time the snapshot was
// taken.
func (h *HistogramSnapshotFloat64) Count() int64 { return h.sample.Count() }

// Max returns the maximum value in the sample at the time the snapshot was
// taken.
func (h *HistogramSnapshotFloat64) Max() float64 { return h.sample.Max() }

// Mean returns the mean of the values in the sample at the time the snapshot
// was taken.
func (h *HistogramSnapshotFloat64) Mean() float64 { return h.sample.Mean() }

// Min returns the minimum value in the sample at the time the snapshot was
// taken.
func (h *HistogramSnapshotFloat64) Min() float64 { return h.sample.Min() }

// Percentile returns an arbitrary percentile of values in the sample at the
// time the snapshot was taken.
func (h *HistogramSnapshotFloat64) Percentile(p float64) float64 {
	return h.sample.Percentile(p)
}

// Percentiles returns a slice of arbitrary percentiles of values in the sample
// at the time the snapshot was taken.
func (h *HistogramSnapshotFloat64) Percentiles(ps []float64) []float64 {
	return h.sample.Percentiles(ps)
}

// Sample returns the Sample underlying the histogram.
func (h *HistogramSnapshotFloat64) Sample() SampleFloat64 { return h.sample }

// Snapshot returns the snapshot.
func (h *HistogramSnapshotFloat64) Snapshot() HistogramFloat64 { return h }

// StdDev returns the standard deviation of the values in the sample at the
// time the snapshot was taken.
func (h *HistogramSnapshotFloat64) StdDev() float64 { return h.sample.StdDev() }

// Sum returns the sum in the sample at the time the snapshot was taken.
func (h *HistogramSnapshotFloat64) Sum() float64 { return h.sample.Sum() }

// Update panics.
func (*HistogramSnapshotFloat64) Update(float64) {
	panic("Update called on a HistogramSnapshotFloat64")
}

// Variance returns the variance of inputs at the time the snapshot was taken.
func (h *HistogramSnapshotFloat64) Variance() float64 { return h.sample.Variance() }

// StandardHistogramFloat64 is the standard implementation of a HistogramFloat64
// and uses a SampleFloat64 to bound its memory use.
type StandardHistogramFloat64 struct {
	sample SampleFloat64
}

// Clear clears the histogram and its sample.
func (h *StandardHistogramFloat64) Clear() { h.sample.Clear() }

// Count returns the number of samples recorded since the histogram was last
// cleared.
func (h *StandardHistogramFloat64) Count() int64 { return h.sample.Count() }

// Max returns the maximum value in the sample.
func (h *StandardHistogramFloat64) Max() float64 { return h.sample.Max() }

// Mean returns the mean of the values in the sample.
func (h *StandardHistogramFloat64) Mean() float64 { return h.sample.Mean() }

// Min returns the minimum value in the sample.
func (h *StandardHistogramFloat64) Min() float64 { return h.sample.Min() }

// Percentile returns an arbitrary percentile of the values in the sample.
func (h *StandardHistogramFloat64) Percentile(p float64) float64 {
	return h.sample.Percentile(p)
}

// Percentiles returns a slice of arbitrary percentiles of the values in the
// sample.
func (h *StandardHistogramFloat64) Percentiles(ps []float64) []float64 {
	return h.sample.Percentiles(ps)
}

// Sample returns the Sample underlying the histogram.
func (h *StandardHistogramFloat64) Sample() SampleFloat64 { return h.sample }

// Snapshot returns a read-only copy of the histogram.
func (h *StandardHistogramFloat64) Snapshot() HistogramFloat64 {
	return &HistogramSnapshotFloat64{sample: h.sample.Snapshot().(*SampleSnapshotFloat64)}
}

// StdDev returns the standard deviation of the values in the sample.
func (h *StandardHistogramFloat64) StdDev() float64 { return h.sample.StdDev() }

// Sum returns the sum in the sample.
func (h *StandardHistogramFloat64) Sum() float64 { return h.sample.Sum() }

// Update samples a new value.
func (h *StandardHistogramFloat64) Update(v float64) { h.sample.Update(v) }

// Variance returns the variance of the values in the sample.
func (h *StandardHistogramFloat64) Variance() float64 { return h.sample.Variance() }
