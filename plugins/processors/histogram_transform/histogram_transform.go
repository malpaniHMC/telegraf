//go:generate ../../../tools/readme_config_includer/generator
package histogram_transform

import (
	_ "embed"
	"strconv"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/processors"
)

//go:embed sample.conf
var sampleConfig string

type Histogram struct {
	FieldName    string    `toml:"field_name"`
	BucketName   string    `toml:"bucket_name"`
	BucketBounds []float32 `toml:"bucket_bounds"`
	Log          telegraf.Logger
	Cache        map[uint64]Buckets
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

func (h *Histogram) ParseFloat(floatString string) float32 {
	float64Val, parse_err := strconv.ParseFloat(floatString, 32)
	if parse_err != nil {
		h.Log.Errorf("Unable to parse to float %v", floatString)

	}
	return float32(float64Val)
}

func (h *Histogram) Apply(in ...telegraf.Metric) []telegraf.Metric {
	h.Log.Errorf("### Length are %d", len(h.Cache))
	h.Log.Errorf("### %d of metrics ", len(in))
	if h.BucketName == "" {
		h.BucketName = "le"
	}

	h.Cache[in[0].HashID()] = Buckets{}

	// h.Log.Error("# of metrics ", len(in))
	// for _, metric := range in {
	// 	if bucketCount, isPresent := metric.GetField(h.FieldName); isPresent {
	// 		h.Log.Debugf("found metric %s with bucket count %v", metric.Name(), bucketCount.(float64))
	// 		if bucketBound, set := metric.GetTag(h.BucketName); set {
	// 			var bucketBoundFloat float32
	// 			if bucketBound == "+Inf" {
	// 				bucketBoundFloat = float32(math.Inf(1))
	// 			} else {
	// 				var err error
	// 				bucketBoundFloat64, err := strconv.ParseFloat(bucketBound, 32)
	// 				if err != nil {
	// 					h.Log.Errorf("Error converting bucket value %s to integer", bucketBound)
	// 				}
	// 				bucketBoundFloat = float32(bucketBoundFloat64)
	// 			}
	// 			buckets[bucketBoundFloat] = int(bucketCount.(float64))
	// 		}
	// 	}
	// }
	// bucketBounds := make([]float32, 0)
	// for bucketVal := range buckets {
	// 	bucketBounds = append(bucketBounds, bucketVal)
	// 	h.Log.Errorf("BucketBounds1: ", bucketVal)
	// }
	// sort.Slice(bucketBounds, func(p, q int) bool {
	// 	return bucketBounds[p] < bucketBounds[q]
	// })
	// for _, bound := range bucketBounds {
	// 	h.Log.Errorf("BucketBounds2: %v", bound)
	// }
	for _, metric := range in {
		if bucketCount, isPresent := metric.GetField(h.FieldName); isPresent {
			// h.Log.Debugf("Editing metric %s with bucket count %v", metric.Name(), bucketCount.(float64))
			bucketCount = int(bucketCount.(float64))
			if bucketBound, set := metric.GetTag(h.BucketName); set {
				buckets, err := h.Cache[metric.HashID()]
				if !err {
					buckets = Buckets{}
				}
				bucketBoundFloat := h.ParseFloat(bucketBound)
				bucketIndex := h.GetIndex(h.BucketBounds, bucketBoundFloat)
				if bucketIndex == -1 {
					h.Log.Errorf("bucket bound %v not found. Something is amiss", bucketBoundFloat)
					continue
				}
				buckets[bucketBoundFloat] = bucketCount.(int)
				// h.Log.Errorf("bucket index of bound %v is %v", bucketBoundFloat, bucketIndex)

				if bucketIndex == 0 {
					// if int(bucketCount.(float64)) != buckets[bucketBoundFloat] {
					// 	h.Log.Errorf("Bucket Counts are not matching. Something is amiss")
					// }
					metric.AddField((h.FieldName + "_disjoint"), bucketCount.(int))
				} else {
					// if int(bucketCount.(float64)) != buckets[bucketBoundFloat] {
					// 	h.Log.Errorf("Bucket Counts are not matching. Something is amiss")
					// }
					prevBucketBound := h.BucketBounds[bucketIndex-1]
					prevBucketCount, prevCached := buckets[prevBucketBound]
					if !prevCached {
						h.Log.Errorf("Unable to calculate disjoint log at the moment")
						continue
					}
					disjointBucketCount := bucketCount.(int) - prevBucketCount
					h.Log.Errorf("Non-first bucket bound %v with disjoint value %v = %v - %v, with bounds %v and %v", bucketBoundFloat, disjointBucketCount, buckets[bucketBoundFloat], buckets[prevBucketBound], bucketBoundFloat, prevBucketBound)
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
		return &Histogram{
			Cache: make(map[uint64]Buckets),
		}
	})
}
