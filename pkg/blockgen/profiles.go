package blockgen

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/thanos-io/thanos/pkg/block/metadata"
	"github.com/thanos-io/thanos/pkg/model"
	"github.com/thanos-io/thanosbench/pkg/seriesgen"
)

type PlanFn func(ctx context.Context, maxTime model.TimeOrDurationValue, extLset labels.Labels, blockEncoder func(BlockSpec) error) error
type ProfileMap map[string]PlanFn

func (p ProfileMap) Keys() (keys []string) {
	for k := range p {
		keys = append(keys, k)
	}
	return keys
}

var (
	Profiles = ProfileMap{
		// Let's say we have 100 applications, 50 metrics each. All rollout every 1h.
		// This makes 2h block to have 15k series, 8h block 45k, 2d block to have 245k series.
		"realistic-k8s-2d-small": realisticK8s([]time.Duration{
			// Two days, from newest to oldest, in the same way Thanos compactor would do.
			2 * time.Hour,
			2 * time.Hour,
			2 * time.Hour,
			8 * time.Hour,
			8 * time.Hour,
			8 * time.Hour,
			8 * time.Hour,
			8 * time.Hour,
			2 * time.Hour,
		}, 1*time.Hour, 100, 50),
		"realistic-k8s-1w-small": realisticK8s([]time.Duration{
			// One week, from newest to oldest, in the same way Thanos compactor would do.
			2 * time.Hour,
			2 * time.Hour,
			2 * time.Hour,
			8 * time.Hour,
			8 * time.Hour,
			48 * time.Hour,
			48 * time.Hour,
			48 * time.Hour,
			2 * time.Hour,
		}, 1*time.Hour, 100, 50),
		"realistic-k8s-30d-tiny": realisticK8s([]time.Duration{
			// 30 days, from newest to oldest.
			2 * time.Hour,
			2 * time.Hour,
			2 * time.Hour,
			8 * time.Hour,
			176 * time.Hour,
			176 * time.Hour,
			176 * time.Hour,
			176 * time.Hour,
			2 * time.Hour,
		}, 1*time.Hour, 1, 5),
		"realistic-k8s-365d-tiny": realisticK8s([]time.Duration{
			// 1y days, from newest to oldest.
			2 * time.Hour,
			2 * time.Hour,
			2 * time.Hour,
			8 * time.Hour,
			176 * time.Hour,
			176 * time.Hour,
			176 * time.Hour,
			176 * time.Hour,
			67 * 24 * time.Hour,
			67 * 24 * time.Hour,
			67 * 24 * time.Hour,
			67 * 24 * time.Hour,
			67 * 24 * time.Hour,
		}, 1*time.Hour, 1, 5),
		"continuous-1w-small": continuous([]time.Duration{
			// One week, from newest to oldest, in the same way Thanos compactor would do.
			2 * time.Hour,
			2 * time.Hour,
			2 * time.Hour,
			8 * time.Hour,
			8 * time.Hour,
			48 * time.Hour,
			48 * time.Hour,
			48 * time.Hour,
			2 * time.Hour,
			// 10,000 series per block.
		}, 100, 100),
		"continuous-30d-tiny": continuous([]time.Duration{
			// 30 days, from newest to oldest.
			2 * time.Hour,
			2 * time.Hour,
			2 * time.Hour,
			8 * time.Hour,
			176 * time.Hour,
			176 * time.Hour,
			176 * time.Hour,
			176 * time.Hour,
			2 * time.Hour,
		}, 1, 5),
		"kruize-15d-tiny": kruize([]time.Duration{
			// 15 days, from newest to oldest.
			2 * time.Hour,
			2 * time.Hour,
			8 * time.Hour,
			8 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			52 * time.Hour,
		}, 1, 1, 8),
		"kruize-15d-1k": kruize([]time.Duration{
			// 15 days, from newest to oldest.
			2 * time.Hour,
			2 * time.Hour,
			8 * time.Hour,
			8 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			52 * time.Hour,
		}, 20, 50, 8),
		"kruize-15d-3k": kruize([]time.Duration{
			// 15 days, from newest to oldest.
			2 * time.Hour,
			2 * time.Hour,
			8 * time.Hour,
			8 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			52 * time.Hour,
		}, 30, 100, 8),
		"kruize-15d-5k": kruize([]time.Duration{
			// 15 days, from newest to oldest.
			2 * time.Hour,
			2 * time.Hour,
			8 * time.Hour,
			8 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			52 * time.Hour,
		}, 50, 100, 8),
		"kruize-15d-10k": kruize([]time.Duration{
			// 15 days, from newest to oldest.
			2 * time.Hour,
			2 * time.Hour,
			8 * time.Hour,
			8 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			72 * time.Hour,
			52 * time.Hour,
		}, 100, 100, 8),
		"continuous-365d-tiny": continuous([]time.Duration{
			// 1y days, from newest to oldest.
			2 * time.Hour,
			2 * time.Hour,
			2 * time.Hour,
			8 * time.Hour,
			176 * time.Hour,
			176 * time.Hour,
			176 * time.Hour,
			176 * time.Hour,
			67 * 24 * time.Hour,
			67 * 24 * time.Hour,
			67 * 24 * time.Hour,
			67 * 24 * time.Hour,
			67 * 24 * time.Hour,
		}, 1, 5),
		"continuous-1w-1series-10000apps": continuous([]time.Duration{
			// One week, from newest to oldest, in the same way Thanos compactor would do.
			2 * time.Hour,
			2 * time.Hour,
			2 * time.Hour,
			8 * time.Hour,
			8 * time.Hour,
			48 * time.Hour,
			48 * time.Hour,
			48 * time.Hour,
			2 * time.Hour,
			// 10,000 series per block.
		}, 10000, 1),
	}
)

func realisticK8s(ranges []time.Duration, rolloutInterval time.Duration, apps int, metricsPerApp int) PlanFn {
	return func(ctx context.Context, maxTime model.TimeOrDurationValue, extLset labels.Labels, blockEncoder func(BlockSpec) error) error {

		// Align timestamps as Prometheus would do.
		maxt := rangeForTimestamp(maxTime.PrometheusTimestamp(), durToMilis(2*time.Hour))

		// Track "rollouts". In heavy used K8s we have rollouts e.g every hour if not more. Account for that.
		lastRollout := maxt - (durToMilis(rolloutInterval) / 2)

		// All our series are gauges.
		common := SeriesSpec{
			Targets: apps,
			Type:    Gauge,
			Characteristics: seriesgen.Characteristics{
				Max:            200000000,
				Min:            10000000,
				Jitter:         30000000,
				ScrapeInterval: 15 * time.Second,
				ChangeInterval: 1 * time.Hour,
			},
		}

		for _, r := range ranges {
			mint := maxt - durToMilis(r) + 1

			b := BlockSpec{
				Meta: metadata.Meta{
					BlockMeta: tsdb.BlockMeta{
						MaxTime:    maxt,
						MinTime:    mint,
						Compaction: tsdb.BlockMetaCompaction{Level: 1},
						Version:    1,
					},
					Thanos: metadata.Thanos{
						Labels:     extLset.Map(),
						Downsample: metadata.ThanosDownsample{Resolution: 0},
						Source:     "blockgen",
					},
				},
			}
			for {
				if ctx.Err() != nil {
					return ctx.Err()
				}

				smaxt := lastRollout + durToMilis(rolloutInterval)
				if smaxt > maxt {
					smaxt = maxt
				}

				smint := lastRollout
				if smint < mint {
					smint = mint
				}

				for i := 0; i < metricsPerApp; i++ {
					s := common

					s.Labels = labels.Labels{
						// TODO(bwplotka): Use different label for metricPerApp cardinality and stable number.
						{Name: "__name__", Value: fmt.Sprintf("k8s_app_metric%d", i)},
						{Name: "next_rollout_time", Value: timestamp.Time(lastRollout).String()},
					}
					s.MinTime = smint
					s.MaxTime = smaxt
					b.Series = append(b.Series, s)
				}

				if lastRollout <= mint {
					break
				}

				lastRollout -= durToMilis(rolloutInterval)
			}

			if err := blockEncoder(b); err != nil {
				return err
			}
			maxt = mint
		}
		return nil
	}
}

func continuous(ranges []time.Duration, apps int, metricsPerApp int) PlanFn {
	return func(ctx context.Context, maxTime model.TimeOrDurationValue, extLset labels.Labels, blockEncoder func(BlockSpec) error) error {

		// Align timestamps as Prometheus would do.
		maxt := rangeForTimestamp(maxTime.PrometheusTimestamp(), durToMilis(2*time.Hour))

		// All our series are gauges.
		common := SeriesSpec{
			Targets: apps,
			Type:    Gauge,
			Characteristics: seriesgen.Characteristics{
				Max:            200000000,
				Min:            10000000,
				Jitter:         30000000,
				ScrapeInterval: 15 * time.Second,
				ChangeInterval: 1 * time.Hour,
			},
		}

		for _, r := range ranges {
			mint := maxt - durToMilis(r) + 1

			if ctx.Err() != nil {
				return ctx.Err()
			}

			b := BlockSpec{
				Meta: metadata.Meta{
					BlockMeta: tsdb.BlockMeta{
						MaxTime:    maxt,
						MinTime:    mint,
						Compaction: tsdb.BlockMetaCompaction{Level: 1},
						Version:    1,
					},
					Thanos: metadata.Thanos{
						Labels:     extLset.Map(),
						Downsample: metadata.ThanosDownsample{Resolution: 0},
						Source:     "blockgen",
					},
				},
			}
			for i := 0; i < metricsPerApp; i++ {
				s := common

				s.Labels = labels.Labels{
					{Name: "__name__", Value: fmt.Sprintf("continuous_app_metric%d", i)},
				}
				s.MinTime = mint
				s.MaxTime = maxt
				b.Series = append(b.Series, s)
			}

			if err := blockEncoder(b); err != nil {
				return err
			}
			maxt = mint
		}
		return nil
	}
}

func kruize(ranges []time.Duration, namespaces int, apps int, metricsPerApp int) PlanFn {
	return func(ctx context.Context, maxTime model.TimeOrDurationValue, extLset labels.Labels, blockEncoder func(BlockSpec) error) error {

		// Metric names
		metrics := [8]string{"container_cpu_usage_seconds_total", "container_cpu_cfs_throttled_seconds_total", "kube_pod_container_resource_limits_cpu",
					"kube_pod_container_resource_requests_cpu", "kube_pod_container_resource_limits_memory", "kube_pod_container_resource_requests_memory",
					"container_memory_working_set_bytes", "container_memory_rss"}
	
		max:= map[string]float64 {"container_cpu_usage_seconds_total": 28, "container_cpu_cfs_throttled_seconds_total": 2, "kube_pod_container_resource_limits_cpu": 32,
                                        "kube_pod_container_resource_requests_cpu":16, "kube_pod_container_resource_limits_memory": 2048, 
					"kube_pod_container_resource_requests_memory": 1024, "container_memory_working_set_bytes": 2000, "container_memory_rss": 512}

		min:= map[string]float64 {"container_cpu_usage_seconds_total": 2, "container_cpu_cfs_throttled_seconds_total": 0, "kube_pod_container_resource_limits_cpu": 4,
                                        "kube_pod_container_resource_requests_cpu":1, "kube_pod_container_resource_limits_memory": 1024, 
					"kube_pod_container_resource_requests_memory": 512, "container_memory_working_set_bytes": 100, "container_memory_rss": 50}
	
		jitter:= map[string]float64 {"container_cpu_usage_seconds_total": 2, "container_cpu_cfs_throttled_seconds_total": 1, "kube_pod_container_resource_limits_cpu": 3,
                                        "kube_pod_container_resource_requests_cpu":2, "kube_pod_container_resource_limits_memory": 20, 
					"kube_pod_container_resource_requests_memory": 10, "container_memory_working_set_bytes": 20, "container_memory_rss": 5}

		// Align timestamps as Prometheus would do.
		maxt := rangeForTimestamp(maxTime.PrometheusTimestamp(), durToMilis(2*time.Hour))

	

		for _, r := range ranges {
			mint := maxt - durToMilis(r) + 1

			if ctx.Err() != nil {
				return ctx.Err()
			}

			b := BlockSpec{
				Meta: metadata.Meta{
					BlockMeta: tsdb.BlockMeta{
						MaxTime:    maxt,
						MinTime:    mint,
						Compaction: tsdb.BlockMetaCompaction{Level: 1},
						Version:    1,
					},
					Thanos: metadata.Thanos{
						Labels:     extLset.Map(),
						Downsample: metadata.ThanosDownsample{Resolution: 0},
						Source:     "blockgen",
					},
				},
			}

			for k := 0; k < namespaces; k++ {
				for j := 0; j < apps; j++ {

					for i := 0; i < metricsPerApp; i++ {
						
						metric := metrics[i]
						max_value := max[metric] * 1000000
						min_value := min[metric] * 1000000
						jitter_value := jitter[metric] * 1000000
	
						// All our series are gauges.
						common := SeriesSpec{
							Targets: 1,
							Type:    Gauge,
							Characteristics: seriesgen.Characteristics{
								Max:            max_value,
								Min:            min_value,
								Jitter:         jitter_value,
								ScrapeInterval: 30 * time.Second,
								ChangeInterval: 1 * time.Hour,
							},
						}

						s := common

						s.Labels = labels.Labels{
							{Name: "__name__", Value: metric},
							{Name: "workload", Value: fmt.Sprintf("tfb-qrh-sample-%d", j)},
							{Name: "workload_type", Value: "deployment"},
							{Name: "container", Value: fmt.Sprintf("tfb-%d", j)},
							{Name: "image", Value: "kruize/tfb-qrh:1.13.2.F_et17"},
							{Name: "namespace", Value: fmt.Sprintf("msc-%d", k)},
						}

						if metrics[i] == "kube_pod_container_resource_limits_cpu" {
							s.Labels = labels.Labels{
								{Name: "__name__", Value: "kube_pod_container_resource_limits"},
								{Name: "workload", Value: fmt.Sprintf("tfb-qrh-sample-%d", j)},
								{Name: "workload_type", Value: "deployment"},
								{Name: "container", Value: fmt.Sprintf("tfb-%d", j)},
								{Name: "workload_type", Value: "deployment"},
								{Name: "image", Value: "kruize/tfb-qrh:1.13.2.F_et17"},
								{Name: "namespace", Value: fmt.Sprintf("msc-%d", k)},
								{Name: "resource", Value: "cpu"},
								{Name: "unit", Value: "core"},
							}
						}

						if metrics[i] == "kube_pod_container_resource_requests_cpu" {
							s.Labels = labels.Labels{
								{Name: "__name__", Value: "kube_pod_container_resource_requests"},
								{Name: "workload", Value: fmt.Sprintf("tfb-qrh-sample-%d", j)},
								{Name: "workload_type", Value: "deployment"},
								{Name: "container", Value: fmt.Sprintf("tfb-%d", j)},
								{Name: "image", Value: "kruize/tfb-qrh:1.13.2.F_et17"},
								{Name: "namespace", Value: fmt.Sprintf("msc-%d", k)},
								{Name: "resource", Value: "cpu"},
								{Name: "unit", Value: "core"},
							}
						}

						if metrics[i] == "kube_pod_container_resource_limits_memory" {
							s.Labels = labels.Labels{
								{Name: "__name__", Value: "kube_pod_container_resource_limits"},
								{Name: "workload", Value: fmt.Sprintf("tfb-qrh-sample-%d", j)},
								{Name: "workload_type", Value: "deployment"},
								{Name: "container", Value: fmt.Sprintf("tfb-%d", j)},
								{Name: "workload_type", Value: "deployment"},
								{Name: "image", Value: "kruize/tfb-qrh:1.13.2.F_et17"},
								{Name: "namespace", Value: fmt.Sprintf("msc-%d", k)},
								{Name: "resource", Value: "memory"},
								{Name: "unit", Value: "bytes"},
							}
						}

						if metrics[i] == "kube_pod_container_resource_requests_memory" {
							s.Labels = labels.Labels{
								{Name: "__name__", Value: "kube_pod_container_resource_requests"},
								{Name: "workload", Value: fmt.Sprintf("tfb-qrh-sample-%d", j)},
								{Name: "workload_type", Value: "deployment"},
								{Name: "container", Value: fmt.Sprintf("tfb-%d", j)},
								{Name: "image", Value: "kruize/tfb-qrh:1.13.2.F_et17"},
								{Name: "namespace", Value: fmt.Sprintf("msc-%d", k)},
								{Name: "resource", Value: "memory"},
								{Name: "unit", Value: "bytes"},
							}
						}
					
						s.MinTime = mint
						s.MaxTime = maxt
						b.Series = append(b.Series, s)
					}
				}
			}

			if err := blockEncoder(b); err != nil {
				return err
			}
			maxt = mint
		}
		return nil
	}
}

func rangeForTimestamp(t int64, width int64) (maxt int64) {
	return (t/width)*width + width
}
