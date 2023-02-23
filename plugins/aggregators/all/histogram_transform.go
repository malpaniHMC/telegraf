//go:build !custom || aggregators || aggregators.histogram_transform

package all

import _ "github.com/influxdata/telegraf/plugins/aggregators/histogram_transform" // register plugin
