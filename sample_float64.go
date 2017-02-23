package metrics

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

type SampleFloat64 interface {
	Clear()
	Count() int64
	Max() float64
	Mean() float64
	Min() float64
	Percentile(float64) float64
	Percentiles([]float64) []float64
	Size() int
	Snapshot() SampleFloat64
	StdDev() float64
	Sum() float64
	Update(float64)
	Values() []float64
	Variance() float64
}

type ExpDecaySampleFloat64 struct {
	alpha         float64
	count         int64
	mutex         sync.Mutex
	reservoirSize int
	t0, t1        time.Time
	values        *expDecaySampleHeapFloat64
}

// NewExpDecaySampleFloat64 constructs a new exponentially-decaying sample with the
// given reservoir size and alpha.
func NewExpDecaySampleFloat64(reservoirSize int, alpha float64) SampleFloat64 {
	s := &ExpDecaySampleFloat64{
		alpha:         alpha,
		reservoirSize: reservoirSize,
		t0:            time.Now(),
		values:        newExpDecaySampleHeapFloat64(reservoirSize),
	}
	s.t1 = s.t0.Add(rescaleThreshold)
	return s
}

// Clear clears all samples.
func (s *ExpDecaySampleFloat64) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count = 0
	s.t0 = time.Now()
	s.t1 = s.t0.Add(rescaleThreshold)
	s.values.Clear()
}

// Count returns the number of samples recorded, which may exceed the
// reservoir size.
func (s *ExpDecaySampleFloat64) Count() int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.count
}

// Max returns the maximum value in the sample, which may not be the maximum
// value ever to be part of the sample.
func (s *ExpDecaySampleFloat64) Max() float64 {
	return SampleMaxFloat64(s.Values())
}

// Mean returns the mean of the values in the sample.
func (s *ExpDecaySampleFloat64) Mean() float64 {
	return SampleMeanFloat64(s.Values())
}

// Min returns the minimum value in the sample, which may not be the minimum
// value ever to be part of the sample.
func (s *ExpDecaySampleFloat64) Min() float64 {
	return SampleMinFloat64(s.Values())
}

// Percentile returns an arbitrary percentile of values in the sample.
func (s *ExpDecaySampleFloat64) Percentile(p float64) float64 {
	return SamplePercentileFloat64(s.Values(), p)
}

// Percentiles returns a slice of arbitrary percentiles of values in the
// sample.
func (s *ExpDecaySampleFloat64) Percentiles(ps []float64) []float64 {
	return SamplePercentilesFloat64(s.Values(), ps)
}

// Size returns the size of the sample, which is at most the reservoir size.
func (s *ExpDecaySampleFloat64) Size() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.values.Size()
}

// Snapshot returns a read-only copy of the sample.
func (s *ExpDecaySampleFloat64) Snapshot() SampleFloat64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	vals := s.values.Values()
	values := make([]float64, len(vals))
	for i, v := range vals {
		values[i] = v.v
	}
	return &SampleSnapshotFloat64{
		count:  s.count,
		values: values,
	}
}

// StdDev returns the standard deviation of the values in the sample.
func (s *ExpDecaySampleFloat64) StdDev() float64 {
	return SampleStdDevFloat64(s.Values())
}

// Sum returns the sum of the values in the sample.
func (s *ExpDecaySampleFloat64) Sum() float64 {
	return SampleSumFloat64(s.Values())
}

// Update samples a new value.
func (s *ExpDecaySampleFloat64) Update(v float64) {
	s.update(time.Now(), v)
}

// Values returns a copy of the values in the sample.
func (s *ExpDecaySampleFloat64) Values() []float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	vals := s.values.Values()
	values := make([]float64, len(vals))
	for i, v := range vals {
		values[i] = v.v
	}
	return values
}

// Variance returns the variance of the values in the sample.
func (s *ExpDecaySampleFloat64) Variance() float64 {
	return SampleVarianceFloat64(s.Values())
}

// update samples a new value at a particular timestamp.  This is a method all
// its own to facilitate testing.
func (s *ExpDecaySampleFloat64) update(t time.Time, v float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count++
	if s.values.Size() == s.reservoirSize {
		s.values.Pop()
	}
	s.values.Push(expDecaySampleFloat64{
		k: math.Exp(t.Sub(s.t0).Seconds()*s.alpha) / rand.Float64(),
		v: v,
	})
	if t.After(s.t1) {
		values := s.values.Values()
		t0 := s.t0
		s.values.Clear()
		s.t0 = t
		s.t1 = s.t0.Add(rescaleThreshold)
		for _, v := range values {
			v.k = v.k * math.Exp(-s.alpha*s.t0.Sub(t0).Seconds())
			s.values.Push(v)
		}
	}
}

// SampleMaxFloat64 returns the maximum value of the slice of float64.
func SampleMaxFloat64(values []float64) float64 {
	if 0 == len(values) {
		return 0
	}
	var max float64 = -math.MaxFloat64
	for _, v := range values {
		if max < v {
			max = v
		}
	}
	return max
}

// SampleMeanFloat64 returns the mean value of the slice of float64.
func SampleMeanFloat64(values []float64) float64 {
	if 0 == len(values) {
		return 0.0
	}
	return float64(SampleSumFloat64(values)) / float64(len(values))
}

// SampleMinFloat64 returns the minimum value of the slice of float64.
func SampleMinFloat64(values []float64) float64 {
	if 0 == len(values) {
		return 0
	}
	var min float64 = math.MaxFloat64
	for _, v := range values {
		if min > v {
			min = v
		}
	}
	return min
}

// SamplePercentilesFloat64 returns an arbitrary percentile of the slice of float64.
func SamplePercentileFloat64(values float64Slice, p float64) float64 {
	return SamplePercentilesFloat64(values, []float64{p})[0]
}

// SamplePercentilesFloat64 returns a slice of arbitrary percentiles of the slice of
// float64.
func SamplePercentilesFloat64(values float64Slice, ps []float64) []float64 {
	scores := make([]float64, len(ps))
	size := len(values)
	if size > 0 {
		sort.Sort(values)
		for i, p := range ps {
			pos := p * float64(size+1)
			if pos < 1.0 {
				scores[i] = float64(values[0])
			} else if pos >= float64(size) {
				scores[i] = float64(values[size-1])
			} else {
				lower := float64(values[int(pos)-1])
				upper := float64(values[int(pos)])
				scores[i] = lower + (pos-math.Floor(pos))*(upper-lower)
			}
		}
	}
	return scores
}

// SampleSnapshotFloat64 is a read-only copy of another Sample.
type SampleSnapshotFloat64 struct {
	count  int64
	values []float64
}

// Clear panics.
func (*SampleSnapshotFloat64) Clear() {
	panic("Clear called on a SampleSnapshotFloat64")
}

// Count returns the count of inputs at the time the snapshot was taken.
func (s *SampleSnapshotFloat64) Count() int64 { return s.count }

// Max returns the maximal value at the time the snapshot was taken.
func (s *SampleSnapshotFloat64) Max() float64 { return SampleMaxFloat64(s.values) }

// Mean returns the mean value at the time the snapshot was taken.
func (s *SampleSnapshotFloat64) Mean() float64 { return SampleMeanFloat64(s.values) }

// Min returns the minimal value at the time the snapshot was taken.
func (s *SampleSnapshotFloat64) Min() float64 { return SampleMinFloat64(s.values) }

// Percentile returns an arbitrary percentile of values at the time the
// snapshot was taken.
func (s *SampleSnapshotFloat64) Percentile(p float64) float64 {
	return SamplePercentileFloat64(s.values, p)
}

// Percentiles returns a slice of arbitrary percentiles of values at the time
// the snapshot was taken.
func (s *SampleSnapshotFloat64) Percentiles(ps []float64) []float64 {
	return SamplePercentilesFloat64(s.values, ps)
}

// Size returns the size of the sample at the time the snapshot was taken.
func (s *SampleSnapshotFloat64) Size() int { return len(s.values) }

// Snapshot returns the snapshot.
func (s *SampleSnapshotFloat64) Snapshot() SampleFloat64 { return s }

// StdDev returns the standard deviation of values at the time the snapshot was
// taken.
func (s *SampleSnapshotFloat64) StdDev() float64 { return SampleStdDevFloat64(s.values) }

// Sum returns the sum of values at the time the snapshot was taken.
func (s *SampleSnapshotFloat64) Sum() float64 { return SampleSumFloat64(s.values) }

// Update panics.
func (*SampleSnapshotFloat64) Update(float64) {
	panic("Update called on a SampleSnapshotFloat64")
}

// Values returns a copy of the values in the sample.
func (s *SampleSnapshotFloat64) Values() []float64 {
	values := make([]float64, len(s.values))
	copy(values, s.values)
	return values
}

// Variance returns the variance of values at the time the snapshot was taken.
func (s *SampleSnapshotFloat64) Variance() float64 { return SampleVarianceFloat64(s.values) }

// SampleStdDevFloat64 returns the standard deviation of the slice of float64.
func SampleStdDevFloat64(values []float64) float64 {
	return math.Sqrt(SampleVarianceFloat64(values))
}

// SampleSumFloat64 returns the sum of the slice of float64.
func SampleSumFloat64(values []float64) float64 {
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum
}

// SampleVarianceFloat64 returns the variance of the slice of float64.
func SampleVarianceFloat64(values []float64) float64 {
	if 0 == len(values) {
		return 0.0
	}
	m := SampleMeanFloat64(values)
	var sum float64
	for _, v := range values {
		d := float64(v) - m
		sum += d * d
	}
	return sum / float64(len(values))
}

// A uniform sample using Vitter's Algorithm R.
//
// <http://www.cs.umd.edu/~samir/498/vitter.pdf>
type UniformSampleFloat64 struct {
	count         int64
	mutex         sync.Mutex
	reservoirSize int
	values        []float64
}

// NewUniformSampleFloat64 constructs a new uniform sample with the given reservoir
// size.
func NewUniformSampleFloat64(reservoirSize int) SampleFloat64 {
	return &UniformSampleFloat64{
		reservoirSize: reservoirSize,
		values:        make([]float64, 0, reservoirSize),
	}
}

// Clear clears all samples.
func (s *UniformSampleFloat64) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count = 0
	s.values = make([]float64, 0, s.reservoirSize)
}

// Count returns the number of samples recorded, which may exceed the
// reservoir size.
func (s *UniformSampleFloat64) Count() int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.count
}

// Max returns the maximum value in the sample, which may not be the maximum
// value ever to be part of the sample.
func (s *UniformSampleFloat64) Max() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return SampleMaxFloat64(s.values)
}

// Mean returns the mean of the values in the sample.
func (s *UniformSampleFloat64) Mean() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return SampleMeanFloat64(s.values)
}

// Min returns the minimum value in the sample, which may not be the minimum
// value ever to be part of the sample.
func (s *UniformSampleFloat64) Min() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return SampleMinFloat64(s.values)
}

// Percentile returns an arbitrary percentile of values in the sample.
func (s *UniformSampleFloat64) Percentile(p float64) float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return SamplePercentileFloat64(s.values, p)
}

// Percentiles returns a slice of arbitrary percentiles of values in the
// sample.
func (s *UniformSampleFloat64) Percentiles(ps []float64) []float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return SamplePercentilesFloat64(s.values, ps)
}

// Size returns the size of the sample, which is at most the reservoir size.
func (s *UniformSampleFloat64) Size() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.values)
}

// Snapshot returns a read-only copy of the sample.
func (s *UniformSampleFloat64) Snapshot() SampleFloat64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	values := make([]float64, len(s.values))
	copy(values, s.values)
	return &SampleSnapshotFloat64{
		count:  s.count,
		values: values,
	}
}

// StdDev returns the standard deviation of the values in the sample.
func (s *UniformSampleFloat64) StdDev() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return SampleStdDevFloat64(s.values)
}

// Sum returns the sum of the values in the sample.
func (s *UniformSampleFloat64) Sum() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return SampleSumFloat64(s.values)
}

// Update samples a new value.
func (s *UniformSampleFloat64) Update(v float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count++
	if len(s.values) < s.reservoirSize {
		s.values = append(s.values, v)
	} else {
		r := rand.Int63n(s.count)
		if r < int64(len(s.values)) {
			s.values[int(r)] = v
		}
	}
}

// Values returns a copy of the values in the sample.
func (s *UniformSampleFloat64) Values() []float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	values := make([]float64, len(s.values))
	copy(values, s.values)
	return values
}

// Variance returns the variance of the values in the sample.
func (s *UniformSampleFloat64) Variance() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return SampleVarianceFloat64(s.values)
}

// expDecaySampleFloat64 represents an individual sample in a heap.
type expDecaySampleFloat64 struct {
	k float64
	v float64
}

func newExpDecaySampleHeapFloat64(reservoirSize int) *expDecaySampleHeapFloat64 {
	return &expDecaySampleHeapFloat64{make([]expDecaySampleFloat64, 0, reservoirSize)}
}

// expDecaySampleHeapFloat64 is a min-heap of expDecaySampleFloat64s.
// The internal implementation is copied from the standard library's container/heap
type expDecaySampleHeapFloat64 struct {
	s []expDecaySampleFloat64
}

func (h *expDecaySampleHeapFloat64) Clear() {
	h.s = h.s[:0]
}

func (h *expDecaySampleHeapFloat64) Push(s expDecaySampleFloat64) {
	n := len(h.s)
	h.s = h.s[0 : n+1]
	h.s[n] = s
	h.up(n)
}

func (h *expDecaySampleHeapFloat64) Pop() expDecaySampleFloat64 {
	n := len(h.s) - 1
	h.s[0], h.s[n] = h.s[n], h.s[0]
	h.down(0, n)

	n = len(h.s)
	s := h.s[n-1]
	h.s = h.s[0 : n-1]
	return s
}

func (h *expDecaySampleHeapFloat64) Size() int {
	return len(h.s)
}

func (h *expDecaySampleHeapFloat64) Values() []expDecaySampleFloat64 {
	return h.s
}

func (h *expDecaySampleHeapFloat64) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !(h.s[j].k < h.s[i].k) {
			break
		}
		h.s[i], h.s[j] = h.s[j], h.s[i]
		j = i
	}
}

func (h *expDecaySampleHeapFloat64) down(i, n int) {
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && !(h.s[j1].k < h.s[j2].k) {
			j = j2 // = 2*i + 2  // right child
		}
		if !(h.s[j].k < h.s[i].k) {
			break
		}
		h.s[i], h.s[j] = h.s[j], h.s[i]
		i = j
	}
}

type float64Slice []float64

func (p float64Slice) Len() int           { return len(p) }
func (p float64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p float64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
