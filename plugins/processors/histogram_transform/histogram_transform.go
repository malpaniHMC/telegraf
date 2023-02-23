//go:generate ../../../tools/readme_config_includer/generator
package histogram_transform

import (
	_ "embed"
	"sort"
	"strconv"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/processors"
)

//go:embed sample.conf
var sampleConfig string

type Histogram struct {
	FieldName         string    `toml:"field_name"`
	BucketName        string    `toml:"bucket_name"`
	BucketBounds      []float32 `toml:"bucket_bounds"`
	Log               telegraf.Logger
	bucketMetricCache map[uint64]telegraf.Metric
}
type Buckets map[float32]int //[BucketBound]BucketCount
func (h *Histogram) GetIndex(slice []float32, n float32) int {
	for i, val := range slice {
		if val == n {
			return i
		}
	}
	return -1
}

func (h *Histogram) Apply(in ...telegraf.Metric) []telegraf.Metric {
	h.Log.Errorf("### Length are %d", len(h.bucketMetricCache))
	h.Log.Errorf("### Got %d metrics ", len(in))

	if h.BucketName == "" {
		h.BucketName = "le"
	}

	// filter metric by field name
	modified := false
	for _, metric := range in {
		if _, exists := metric.GetField(h.FieldName); !exists {
			h.Log.Errorf("Not Processing %s %v", metric.Name(), metric.FieldList())
			continue
		}

		h.Log.Errorf("Processing %s %v", metric.Name(), metric.FieldList())
		h.bucketMetricCache[metric.HashID()] = metric
		metric.Drop()
		modified = true
	}

	if !modified {
		return in
	}

	if len(h.bucketMetricCache) != len(h.BucketBounds)+1 {
		h.Log.Error("Don't have enough buckets", len(h.bucketMetricCache), len(h.BucketBounds))
		return in
	}

	sortedByBound := make([]telegraf.Metric, 0)
	for _, metric := range h.bucketMetricCache {
		sortedByBound = append(sortedByBound, metric)
	}

	sort.Slice(sortedByBound, func(i, j int) bool {
		bucketBound1Str, _ := sortedByBound[i].GetTag(h.BucketName)
		bucketBound2Str, _ := sortedByBound[j].GetTag(h.BucketName)

		if bucketBound1Str == "+Inf" {
			return true
		}

		if bucketBound2Str == "+Inf" {
			return false
		}

		bucketBound1, _ := strconv.ParseFloat(bucketBound1Str, 64)
		bucketBound2, _ := strconv.ParseFloat(bucketBound2Str, 64)
		return bucketBound1 > bucketBound2
	})

	disjointMetricBuckets := h.calculateDisjointBuckets(sortedByBound)
	h.bucketMetricCache = make(map[uint64]telegraf.Metric)
	return append(in, disjointMetricBuckets...)
}

func (h *Histogram) calculateDisjointBuckets(cumulativeMetrics []telegraf.Metric) (disjointMetrics []telegraf.Metric) {
	for bucketIdx, bucket := range cumulativeMetrics {
		metric := bucket.Copy()
		metricValue, _ := metric.GetField(h.FieldName)
		tag, _ := metric.GetTag(h.BucketName)
		if bucketIdx == len(cumulativeMetrics)-1 {
			h.Log.Errorf("Processing the last bucket for %s, value: %v %s", metric.Name(), metricValue, tag)
			metric.AddField(h.FieldName+"_disjoint", metricValue)
			// metric.RemoveField(h.FieldName)
			disjointMetrics = append(disjointMetrics, metric)
			continue
		}

		previousValue, _ := cumulativeMetrics[bucketIdx+1].GetField(h.FieldName)
		previousTag, _ := cumulativeMetrics[bucketIdx+1].GetTag(h.BucketName)

		h.Log.Errorf("Processing the bucket for %s, value: %v %s, previous bucket: %v %s", metric.Name(), metricValue, tag, previousValue, previousTag)
		metric.AddField(h.FieldName+"_disjoint", convertToFloat(metricValue)-convertToFloat(previousValue))
		// metric.RemoveField(h.FieldName)
		disjointMetrics = append(disjointMetrics, metric)
	}

	return disjointMetrics
}

func convertToFloat(value interface{}) float64 {
	return value.(float64)
}

func (*Histogram) SampleConfig() string {
	return sampleConfig
}

func init() {
	processors.Add("histogram_transform", func() telegraf.Processor {
		return &Histogram{
			bucketMetricCache: make(map[uint64]telegraf.Metric),
		}
	})
}
