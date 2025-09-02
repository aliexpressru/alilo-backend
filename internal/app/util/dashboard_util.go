package util

import (
	"fmt"

	models "github.com/aliexpressru/alilo-backend/internal/app/dbmodels"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
	"github.com/grafana/grafana-foundation-sdk/go/timeseries"
)

const (
	RTPrefix      = "RT"
	ErrPrefix     = "Err"
	RPSPrefix     = "RPS"
	DefaultHeight = 6
)

type Dashboard struct {
	*dashboard.DashboardBuilder
}

type Panel struct {
	Title       string
	Description string
	Unit        string
	Expr        string
	GridPos     dashboard.GridPos
}

type ScriptsSlice struct {
	models.SimpleScriptSlice
	models.ScriptSlice
}

func NewDashBoardBuilder(title string, description string) (*Dashboard, error) {
	uid, err := RandomMixedCaseString(40)
	if err != nil {
		return nil, err
	}

	builder := dashboard.NewDashboardBuilder(title).
		Uid(uid).
		Tags([]string{"response time", "requests per minutes", "errors count"}).
		Refresh("1m").
		Time("now-30m", "now").
		Timezone(common.TimeZoneBrowser).
		Description(description)

	return &Dashboard{builder}, nil
}

// WithTimeSeriesPanel add timeSeries panel
func (builder *Dashboard) WithTimeSeriesPanel(panel *Panel) *Dashboard {
	builder.
		WithPanel(
			timeseries.NewPanelBuilder().
				Title(panel.Title).
				Description(panel.Description).
				Unit(panel.Unit).
				Min(0).
				WithTarget(
					prometheus.NewDataqueryBuilder().
						Expr(panel.Expr),
				).
				GridPos(panel.GridPos),
		)

	return builder
}

func (builder *Dashboard) SimpleScriptTimeSeriesRpsPanel(script *models.SimpleScript, x uint32, y uint32) *Dashboard {
	return builder.WithTimeSeriesPanel(
		&Panel{
			Title:       fmt.Sprintf("%s %s", RPSPrefix, script.Name),
			Description: script.Description,
			Unit:        "",
			Expr:        script.ExprRPS,
			GridPos: dashboard.GridPos{
				H: DefaultHeight,
				W: 8,
				X: x,
				Y: y,
			},
		},
	)
}

func (builder *Dashboard) SimpleScriptTimeSeriesRTPanel(script *models.SimpleScript, x uint32, y uint32) *Dashboard {
	return builder.WithTimeSeriesPanel(
		&Panel{
			Title:       fmt.Sprintf("%s %s", RTPrefix, script.Name),
			Description: script.Description,
			Unit:        "",
			Expr:        script.ExprRT,
			GridPos: dashboard.GridPos{
				H: 6,
				W: 8,
				X: x,
				Y: y,
			},
		},
	)
}

func (builder *Dashboard) SimpleScriptTimeSeriesErrPanel(script *models.SimpleScript, x uint32, y uint32) *Dashboard {
	return builder.WithTimeSeriesPanel(
		&Panel{
			Title:       fmt.Sprintf("%s %s", ErrPrefix, script.Name),
			Description: script.Description,
			Unit:        "",
			Expr:        script.ExprErr,
			GridPos: dashboard.GridPos{
				H: DefaultHeight,
				W: 8,
				X: x,
				Y: y,
			},
		},
	)
}

func (builder *Dashboard) ScriptTimeSeriesRpsPanel(script *models.Script, x uint32, y uint32) *Dashboard {
	return builder.WithTimeSeriesPanel(
		&Panel{
			Title:       fmt.Sprintf("%s %s", RPSPrefix, script.Name),
			Description: script.Descrip.String,
			Unit:        "",
			Expr:        script.ExprRPS,
			GridPos: dashboard.GridPos{
				H: 6,
				W: 8,
				X: x,
				Y: y,
			},
		},
	)
}

func (builder *Dashboard) ScriptTimeSeriesRTPanel(script *models.Script, x uint32, y uint32) *Dashboard {
	return builder.WithTimeSeriesPanel(
		&Panel{
			Title:       fmt.Sprintf("%s %s", RTPrefix, script.Name),
			Description: script.Descrip.String,
			Unit:        "",
			Expr:        script.ExprRT,
			GridPos: dashboard.GridPos{
				H: 6,
				W: 8,
				X: x,
				Y: y,
			},
		},
	)
}

func (builder *Dashboard) ScriptTimeSeriesErrPanel(script *models.Script, x uint32, y uint32) *Dashboard {
	return builder.WithTimeSeriesPanel(
		&Panel{
			Title:       fmt.Sprintf("%s %s", ErrPrefix, script.Name),
			Description: script.Descrip.String,
			Unit:        "",
			Expr:        script.ExprErr,
			GridPos: dashboard.GridPos{
				H: DefaultHeight,
				W: 8,
				X: x,
				Y: y,
			},
		},
	)
}
