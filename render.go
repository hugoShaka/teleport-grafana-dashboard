package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
	"github.com/grafana/grafana-foundation-sdk/go/statetimeline"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"
	"github.com/grafana/grafana-foundation-sdk/go/units"
)

func main() {
	release := releaseInfo{
		namespace: "teleport-cluster",
		name:      "teleport",
	}

	releaseHash := sha256.Sum256([]byte(release.namespace + "/" + release.name))
	releaseId := hex.EncodeToString(releaseHash[:])[:8]

	builder := dashboard.NewDashboardBuilder("Teleport / Base Kubernetes").
		Uid("teleport-kubernetes-base-"+releaseId).
		Tags([]string{"teleport", "kubernetes"}).
		Refresh("1m").
		Time("now-1h", "now").
		Timezone(common.TimeZoneBrowser)

	authRow := oneRow(7, authCPUPanel(release), authMemoryPanel(release), authNetworkPanel(release))
	for _, panel := range authRow {
		builder.WithPanel(panel.Legend(noLegend()).Tooltip(defaultTooltip()))
	}

	proxyRow := oneRow(7, proxyCPUPanel(release), proxyMemoryPanel(release), proxyNetworkPanel(release))
	for _, panel := range proxyRow {
		builder.WithPanel(panel.Legend(noLegend()).Tooltip(defaultTooltip()))
	}

	builder.WithPanel(authStatusPanel(release))
	builder.WithPanel(proxyStatusPanel(release))

	sampleDashboard, err := builder.Build()
	if err != nil {
		panic(err)
	}

	dashboardJson, err := json.MarshalIndent(sampleDashboard, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(dashboardJson))
}

type releaseInfo struct {
	namespace string
	name      string
}

func (r releaseInfo) authPodSelector() string {
	return fmt.Sprintf(`namespace="%s", pod=~"%s-auth-.*"`, r.namespace, r.name)
}

func (r releaseInfo) proxyPodSelector() string {
	return fmt.Sprintf(`namespace="%s", pod=~"%s-proxy-.*"`, r.namespace, r.name)
}

func podMemoryRatioQuery(selector string) string {
	return podMemoryQuery(selector) + " / " + podMemoryRequestQuery(selector)
}

func podMemoryQuery(selector string) string {
	return fmt.Sprintf(`max by (pod) (container_memory_working_set_bytes{%s, container="teleport"})`, selector)
}

func podMemoryRequestQuery(selector string) string {
	return fmt.Sprintf(`min by (pod) (kube_pod_container_resource_requests{%s, container="teleport", resource="memory"})`, selector)
}

func podCPURatioQuery(selector string) string {
	return podCPUQuery(selector) + " / " + podCPURequestQuery(selector)
}

func podCPUQuery(selector string) string {
	return fmt.Sprintf(`max by (pod) (rate(container_cpu_usage_seconds_total{%s, container="teleport"}[1m]))`, selector)
}

func podCPURequestQuery(selector string) string {
	return fmt.Sprintf(`min by (pod) (kube_pod_container_resource_requests{%s, container="teleport", resource="cpu"})`, selector)
}

func podNetworkRxQuery(selector string) string {
	return fmt.Sprintf(`max by (pod) (rate(container_network_receive_bytes_total{%s}[1m]))`, selector)
}

func podNetworkTxQuery(selector string) string {
	return fmt.Sprintf(`max by (pod) (rate(container_network_transmit_bytes_total{%s}[1m]))`, selector)
}

func oneRow(height int, panels ...*timeseries.PanelBuilder) []*timeseries.PanelBuilder {
	widthPanel := 24 / len(panels)
	result := make([]*timeseries.PanelBuilder, len(panels))
	for i, panel := range panels {
		result[i] = panel.GridPos(dashboard.GridPos{
			H:      uint32(height),
			W:      uint32(widthPanel),
			X:      uint32(i * widthPanel),
			Y:      0,
			Static: nil,
		})
	}
	return result
}

func authMemoryPanel(release releaseInfo) *timeseries.PanelBuilder {
	query := podMemoryRatioQuery(release.authPodSelector())
	target := prometheus.NewDataqueryBuilder().Expr(query).LegendFormat("{{ pod }}")
	return timeseries.NewPanelBuilder().
		Title("Auth Memory Usage").
		Unit(units.PercentUnit).
		AxisSoftMin(0).
		AxisSoftMax(1).
		WithTarget(target)
}

func proxyMemoryPanel(release releaseInfo) *timeseries.PanelBuilder {
	query := podMemoryRatioQuery(release.proxyPodSelector())
	target := prometheus.NewDataqueryBuilder().Expr(query).LegendFormat("{{ pod }}")
	return timeseries.NewPanelBuilder().
		Title("Proxy Memory Usage").
		Unit(units.PercentUnit).
		AxisSoftMin(0).
		AxisSoftMax(1).
		WithTarget(target)
}

func authCPUPanel(release releaseInfo) *timeseries.PanelBuilder {
	query := podCPURatioQuery(release.authPodSelector())
	target := prometheus.NewDataqueryBuilder().Expr(query).LegendFormat("{{ pod }}")
	return timeseries.NewPanelBuilder().
		Title("Auth CPU Usage").
		Unit(units.PercentUnit).
		AxisSoftMin(0).
		AxisSoftMax(1).
		WithTarget(target)
}

func proxyCPUPanel(release releaseInfo) *timeseries.PanelBuilder {
	query := podCPURatioQuery(release.proxyPodSelector())
	target := prometheus.NewDataqueryBuilder().Expr(query).LegendFormat("{{ pod }}")
	return timeseries.NewPanelBuilder().
		Title("Proxy CPU Usage").
		Unit(units.PercentUnit).
		AxisSoftMin(0).
		AxisSoftMax(1).
		WithTarget(target)
}

func authNetworkPanel(release releaseInfo) *timeseries.PanelBuilder {
	queryRx := podNetworkRxQuery(release.authPodSelector())
	queryTx := podNetworkTxQuery(release.authPodSelector())
	targetRx := prometheus.NewDataqueryBuilder().Expr(queryRx).LegendFormat("{{ pod }}-rx")
	targetTx := prometheus.NewDataqueryBuilder().Expr(queryTx).LegendFormat("{{ pod }}-tx")
	return timeseries.NewPanelBuilder().
		Title("Auth Network Usage").
		Unit(units.BytesPerSecondSI).
		AxisSoftMin(0).
		WithTarget(targetRx).
		WithTarget(targetTx)
}

func proxyNetworkPanel(release releaseInfo) *timeseries.PanelBuilder {
	queryRx := podNetworkRxQuery(release.proxyPodSelector())
	queryTx := podNetworkTxQuery(release.proxyPodSelector())
	targetRx := prometheus.NewDataqueryBuilder().Expr(queryRx).LegendFormat("{{ pod }}-rx")
	targetTx := prometheus.NewDataqueryBuilder().Expr(queryTx).LegendFormat("{{ pod }}-tx")
	return timeseries.NewPanelBuilder().
		Title("Proxy Network Usage").
		Unit(units.BytesPerSecondSI).
		AxisSoftMin(0).
		WithTarget(targetRx).
		WithTarget(targetTx)
}

func podStatusQuery(selector string) string {
	return fmt.Sprintf(
		"kube_pod_status_ready{%s, condition=\"false\"} * 100 + ignoring(condition) kube_pod_status_ready{%s, condition=\"true\"} * 200 + ignoring (condition) kube_pod_status_ready{%s, condition=\"unknown\"} * 1",
		selector, selector, selector,
	)
}

func ptr[T any](val T) *T {
	return &val
}

func statusPanelMappings() []dashboard.ValueMapping {
	unknownRange := dashboard.NewRangeMap()
	unknownRange.Options.From = ptr[float64](0)
	unknownRange.Options.To = ptr[float64](99)
	unknownRange.Options.Result.Color = ptr("purple")
	unknownRange.Options.Result.Text = ptr("Unknown")

	notReadyRange := dashboard.NewRangeMap()
	notReadyRange.Options.From = ptr[float64](99)
	notReadyRange.Options.To = ptr[float64](199)
	notReadyRange.Options.Result.Color = ptr("red")
	notReadyRange.Options.Result.Text = ptr("NotReady")

	readyRange := dashboard.NewRangeMap()
	readyRange.Options.From = ptr[float64](199)
	readyRange.Options.To = ptr[float64](299)
	readyRange.Options.Result.Color = ptr("green")
	readyRange.Options.Result.Text = ptr("Ready")

	unknown := dashboard.NewValueMapping()
	unknown.RangeMap = unknownRange

	notReady := dashboard.NewValueMapping()
	notReady.RangeMap = notReadyRange

	ready := dashboard.NewValueMapping()
	ready.RangeMap = readyRange

	return []dashboard.ValueMapping{
		*unknown,
		*notReady,
		*ready,
	}
}

func authStatusPanel(release releaseInfo) *statetimeline.PanelBuilder {
	query := podStatusQuery(release.authPodSelector())
	target := prometheus.NewDataqueryBuilder().Expr(query).LegendFormat("{{ pod }}")
	return statetimeline.NewPanelBuilder().
		Title("Auth pod status").
		WithTarget(target).Mappings(statusPanelMappings()).GridPos(
		dashboard.GridPos{
			H: 7,
			W: 24,
		}).Description("Represents the Teleport pods status. A pod can be unready for a few minutes on startup, or when being terminated. However a pod turning unready during normal usage is unexpected and indicates an issue.").
		ShowValue(common.VisibilityModeNever)
}

func proxyStatusPanel(release releaseInfo) *statetimeline.PanelBuilder {
	query := podStatusQuery(release.proxyPodSelector())
	target := prometheus.NewDataqueryBuilder().Expr(query).LegendFormat("{{ pod }}")
	return statetimeline.NewPanelBuilder().
		Title("Proxy pods status").
		WithTarget(target).Mappings(statusPanelMappings()).GridPos(
		dashboard.GridPos{
			H: 7,
			W: 24,
		}).Description("Represents the Teleport pods status. A pod can be unready for a few minutes on startup, or when being terminated. However a pod turning unready during normal usage is unexpected and indicates an issue.").
		ShowValue(common.VisibilityModeNever)
}

func noLegend() *common.VizLegendOptionsBuilder {
	legend := common.NewVizLegendOptionsBuilder()
	legend.ShowLegend(false)
	return legend
}

func defaultTooltip() *common.VizTooltipOptionsBuilder {
	tooltip := common.NewVizTooltipOptionsBuilder()
	tooltip.Mode(common.TooltipDisplayModeMulti)
	tooltip.HideZeros(true)
	tooltip.Sort(common.SortOrderDescending)
	return tooltip
}
