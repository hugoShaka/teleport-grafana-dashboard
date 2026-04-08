package main

import (
	"fmt"

	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
	"github.com/grafana/grafana-foundation-sdk/go/statetimeline"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"
	"github.com/grafana/grafana-foundation-sdk/go/units"
)

func MemoryDashboard(release releaseInfo) *dashboard.DashboardBuilder {
	builder := dashboard.NewDashboardBuilder("Teleport / Memory Usage").
		Uid("teleport-memory-"+release.id()).
		Tags([]string{"teleport", "memory"}).
		Refresh("1m").
		Time("now-1h", "now").
		Timezone(common.TimeZoneBrowser)

	// variables
	podVar := dashboard.NewQueryVariableBuilder("pod").
		Description("Name of the pod to observe").
		Refresh(dashboard.VariableRefreshOnTimeRangeChanged).
		Query(dashboard.StringOrMap{
			String: ptr("label_values(go_memstats_alloc_bytes{container=\"teleport\", namespace=\"teleport-cluster\"},pod)"),
		})
	builder.WithVariable(podVar)
	builder.WithPanel(podStatusPanel(release))

	// memory
	// cadvisor, go
	builder.WithPanel(podMemoryCadvisorPanel(release))
	builder.WithPanel(podMemoryGoHeapPanel(release))

	// runtime
	// GC, objects

	//

	return builder
}

func podSelector(pod, namespace string) string {
	return fmt.Sprintf(`namespace="%s", pod="%s"`, namespace, pod)
}

const podVariable = "$pod"

func podStatusPanel(release releaseInfo) *statetimeline.PanelBuilder {
	query := podStatusQuery(podSelector(podVariable, release.namespace))
	target := prometheus.NewDataqueryBuilder().Expr(query).LegendFormat("{{ pod }}")
	return statetimeline.NewPanelBuilder().
		Title("Pod status").
		WithTarget(target).Mappings(statusPanelMappings()).GridPos(
		dashboard.GridPos{
			H: 3,
			W: 24,
		}).Description("Represents the Teleport pods status. A pod can be unready for a few minutes on startup, or when being terminated. However a pod turning unready during normal usage is unexpected and indicates an issue.").
		ShowValue(common.VisibilityModeNever).PerPage(0).Legend(noLegend())
}

func podRSSQuery(selector string) string {
	return fmt.Sprintf(`max by (pod) (container_memory_rss{%s, container="teleport"})`, selector)
}

func podMemoryCadvisorPanel(release releaseInfo) *timeseries.PanelBuilder {
	selector := podSelector(podVariable, release.namespace)
	memoryRequestQuery := podMemoryRequestQuery(selector)
	memoryRequestTarget := prometheus.NewDataqueryBuilder().Expr(memoryRequestQuery).LegendFormat("request")
	wssQuery := podMemoryQuery(selector)
	wssTarget := prometheus.NewDataqueryBuilder().Expr(wssQuery).LegendFormat("WSS")
	rssQuery := podRSSQuery(selector)
	rssTarget := prometheus.NewDataqueryBuilder().Expr(rssQuery).LegendFormat("RSS")

	return timeseries.NewPanelBuilder().
		Title("Container Memory").
		Unit(units.BytesIEC).
		AxisSoftMin(0).
		WithTarget(memoryRequestTarget).
		WithTarget(wssTarget).
		WithTarget(rssTarget)
}

func podHeapInUseQuery(selector string) string {
	return fmt.Sprintf(`max by (pod) (go_memstats_heap_inuse_bytes{%s, container="teleport"})`, selector)
}
func podHeapReleasedQuery(selector string) string {
	return fmt.Sprintf(`max by (pod) (go_memstats_heap_released_bytes{%s, container="teleport"})`, selector)
}
func podHeapIdleQuery(selector string) string {
	return fmt.Sprintf(`max by (pod) (go_memstats_heap_idle_bytes{%s, container="teleport"})`, selector)
}

func podMemoryGoHeapPanel(release releaseInfo) *timeseries.PanelBuilder {
	selector := podSelector(podVariable, release.namespace)
	heapInUseQuery := podHeapInUseQuery(selector)
	heapInUseTarget := prometheus.NewDataqueryBuilder().Expr(heapInUseQuery).LegendFormat("in use")
	heapIdleQuery := podHeapIdleQuery(selector)
	heapIdleTarget := prometheus.NewDataqueryBuilder().Expr(heapIdleQuery).LegendFormat("idle")
	heapReleasedQuery := podHeapReleasedQuery(selector)
	heapReleasedTarget := prometheus.NewDataqueryBuilder().Expr(heapReleasedQuery).LegendFormat("released")
	return timeseries.NewPanelBuilder().
		Title("Go Heap").
		Unit(units.BytesIEC).
		AxisSoftMin(0).
		WithTarget(heapInUseTarget).
		WithTarget(heapIdleTarget).
		WithTarget(heapReleasedTarget).FillOpacity(20)
}

// Note: I'd love to expose per-pod PSI but the metric rquires cadvisor 0.52 in
// kube 1.33 and above.
