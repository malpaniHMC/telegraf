//go:generate ../../../tools/readme_config_includer/generator
package histogram_transform

// histogram_transform.go

import (
	_ "embed"
	"strconv"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/aggregators"
)

//go:embed sample.conf
var sampleConfig string

type Histogram_Transform struct {
	// caches for metric fields, names, and tags
	FieldName    string    `toml:"field_name"`
	BucketName   string    `toml:"bucket_name"`
	BucketBounds []float32 `toml:"bucket_bounds"`
	Log          telegraf.Logger
	bucketCache  map[uint64]map[int]int
}

func NewMin() telegraf.Aggregator {
	h := &Histogram_Transform{}
	h.Reset()
	return h
}

func (*Histogram_Transform) SampleConfig() string {
	return sampleConfig
}

func (h *Histogram_Transform) Init() error {
	return nil
}

func (h *Histogram_Transform) ParseInt(intString string) int {
	intVal, parse_err := strconv.Atoi(intString)
	if parse_err != nil {
		h.Log.Errorf("Unable to parse to float %v", intString)

	}
	return intVal
}

func (h *Histogram_Transform) Add(in telegraf.Metric) {
	id := in.HashID()
	if _, ok := h.bucketCache[id]; !ok {
		h.bucketCache[id] = make(map[int]int)
	}
	if bucketCount, isPresent := in.GetField(h.FieldName); isPresent {
		bucketCount = int(bucketCount.(float64))
		if bucketBound, set := in.GetTag(h.BucketName); set {
			bucketBoundInt := h.ParseInt(bucketBound)
			h.bucketCache[id][bucketBoundInt] = bucketCount.(int)
		}
	}
}

func (h *Histogram_Transform) Push(acc telegraf.Accumulator) {

}

func (h *Histogram_Transform) Reset() {
	// h.bucketCache = make(map[uint64]map[int]int)
	// h.nameCache = make(map[uint64]string)
	// h.tagCache = make(map[uint64]map[string]string)
}

func convert(in interface{}) (float64, bool) {
	switch v := in.(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

func init() {
	aggregators.Add("histogram_transform", func() telegraf.Aggregator {
		return NewMin()
	})
}
