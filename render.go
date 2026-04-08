package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"
)

func main() {
	outputDir := "./out/"
	release := releaseInfo{
		namespace: "cloud-gravitational-io-hugoshaka-internal",
		name:      "teleport",
	}

	dashboards := []*dashboard.DashboardBuilder{
		KubeDashboard(release),
		MemoryDashboard(release),
	}

	for _, builder := range dashboards {
		d, err := builder.Build()
		if err != nil {
			panic(err)
		}

		out, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			panic(err)
		}

		file := filepath.Join(outputDir, *d.Uid+".json")
		if err := os.WriteFile(file, out, 0644); err != nil {
			panic(err)
		}
		fmt.Printf("Wrote %s\n", file)
	}
}

type releaseInfo struct {
	namespace string
	name      string
}

func (r releaseInfo) id() string {
	releaseHash := sha256.Sum256([]byte(r.namespace + "/" + r.name))
	return hex.EncodeToString(releaseHash[:])[:8]
}

func (r releaseInfo) authPodSelector() string {
	return fmt.Sprintf(`namespace="%s", pod=~"%s-auth-.*"`, r.namespace, r.name)
}

func (r releaseInfo) proxyPodSelector() string {
	return fmt.Sprintf(`namespace="%s", pod=~"%s-proxy-.*"`, r.namespace, r.name)
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

func ptr[T any](val T) *T {
	return &val
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
