//go:generate ../../../tools/readme_config_includer/generator
package histogram_transform

import (
	_ "embed"
	"math"
	"sort"
	"strconv"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/processors"
)

//go:embed sample.conf
var sampleConfig string

type Histogram struct {
	FieldName  string `toml:"field_name"`
	BucketName string `toml:"bucket_name"`
	Log        telegraf.Logger
}

func GetIndex(slice []float64, n float64) int {
	for i, val := range slice {
		if val == n {
			return i
		}
	}
	return -1
}

func (h *Histogram) Apply(in ...telegraf.Metric) []telegraf.Metric {
	buckets := map[float64]int{} //[BucketBound]BucketCount
	if h.BucketName == "" {
		h.BucketName = "le"
	}
	for _, metric := range in {
		if bucketCount, isPresent := metric.GetField(h.FieldName); isPresent {
			h.Log.Debugf("found metric %s with bucket count %v", metric.Name(), bucketCount.(float64))
			if bucketBound, set := metric.GetTag(h.BucketName); set {
				var bucketBoundFloat float64
				if bucketBound == "+Inf" {
					bucketBoundFloat = math.Inf(1)
				} else {
					var err error
					bucketBoundFloat, err = strconv.ParseFloat(bucketBound, 64)
					if err != nil {
						h.Log.Errorf("Error converting bucket value %s to integer", bucketBound)
					}
				}

				buckets[bucketBoundFloat] = int(bucketCount.(float64))
			}
		}
	}
	bucketBounds := make([]float64, len(buckets))
	for bucketVal := range buckets {
		bucketBounds = append(bucketBounds, bucketVal)
	}
	sort.Slice(bucketBounds, func(p, q int) bool {
		return bucketBounds[p] < bucketBounds[q]
	})

	for _, metric := range in {
		if bucketCount, isPresent := metric.GetField(h.FieldName); isPresent {
			h.Log.Debugf("Editing metric %s with bucket count %v", metric.Name(), bucketCount.(float64))
			if bucketBound, set := metric.GetTag(h.BucketName); set {
				bucketBoundFloat, err := strconv.ParseFloat(bucketBound, 64)
				if err != nil {
					h.Log.Errorf("Unable to parse to float %v", bucketBound)
					continue
				}

				bucketIndex := GetIndex(bucketBounds, bucketBoundFloat)
				if bucketIndex == 0 {
					if int(bucketCount.(float64)) != buckets[bucketBoundFloat] {
						h.Log.Errorf("Bucket Counts are not matching. Something is amiss")
					}
					metric.AddField((h.FieldName + "_disjoint"), buckets[bucketBoundFloat])
				} else {
					if int(bucketCount.(float64)) != buckets[bucketBoundFloat] {
						h.Log.Errorf("Bucket Counts are not matching. Something is amiss")
					}
					prevBucketBoundIndex := bucketBounds[bucketIndex-1]
					disjointBucketCount := buckets[bucketBoundFloat] - buckets[prevBucketBoundIndex]
					metric.AddField((h.FieldName + "_disjoint"), disjointBucketCount)
				}
			}
		}
	}
	return in
}

func (*Histogram) SampleConfig() string {
	return sampleConfig
}

func init() {
	processors.Add("histogram_transform", func() telegraf.Processor {
		return &Histogram{}
	})
}
