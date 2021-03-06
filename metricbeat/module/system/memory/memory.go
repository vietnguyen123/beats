// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build darwin freebsd linux openbsd windows

package memory

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	mem "github.com/elastic/beats/v7/libbeat/metric/system/memory"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	linux "github.com/elastic/beats/v7/metricbeat/module/linux/memory"
	"github.com/elastic/beats/v7/metricbeat/module/system"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "memory", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching system memory metrics.
type MetricSet struct {
	mb.BaseMetricSet
	IsAgent bool
}

// New is a mb.MetricSetFactory that returns a memory.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	systemModule, ok := base.Module().(*system.Module)
	if !ok {
		return nil, fmt.Errorf("unexpected module type")
	}

	return &MetricSet{BaseMetricSet: base, IsAgent: systemModule.IsAgent}, nil
}

// Fetch fetches memory metrics from the OS.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	memStat, err := mem.Get()
	if err != nil {
		return errors.Wrap(err, "memory")
	}
	mem.AddMemPercentage(memStat)

	swapStat, err := mem.GetSwap()
	if err != nil {
		return errors.Wrap(err, "swap")
	}
	mem.AddSwapPercentage(swapStat)

	memory := common.MapStr{
		"total": memStat.Total,
		"used": common.MapStr{
			"bytes": memStat.Used,
			"pct":   memStat.UsedPercent,
		},
		"free": memStat.Free,
		"actual": common.MapStr{
			"free": memStat.ActualFree,
			"used": common.MapStr{
				"pct":   memStat.ActualUsedPercent,
				"bytes": memStat.ActualUsed,
			},
		},
	}

	vmstat, err := mem.GetVMStat()
	if err != nil {
		return errors.Wrap(err, "VMStat")
	}

	swap := common.MapStr{
		"total": swapStat.Total,
		"used": common.MapStr{
			"bytes": swapStat.Used,
			"pct":   swapStat.UsedPercent,
		},
		"free": swapStat.Free,
	}

	if vmstat != nil {
		// Swap in and swap out numbers
		swap["in"] = common.MapStr{
			"pages": vmstat.Pswpin,
		}
		swap["out"] = common.MapStr{
			"pages": vmstat.Pswpout,
		}
		//Swap readahead
		//See https://www.kernel.org/doc/ols/2007/ols2007v2-pages-273-284.pdf
		swap["readahead"] = common.MapStr{
			"pages":  vmstat.SwapRa,
			"cached": vmstat.SwapRaHit,
		}
	}

	// for backwards compatibility, only report if we're not in fleet mode
	if !m.IsAgent {
		err := linux.FetchLinuxMemStats(memory)
		if err != nil {
			return errors.Wrap(err, "error getting page stats")
		}
	}

	memory["swap"] = swap

	r.Event(mb.Event{
		MetricSetFields: memory,
	})

	return nil
}
