//go:build desktop

package main

import (
	"context"
	"encoding/json"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const desktopCodeSummaryEvent = "gopanel:code-summary"

type desktopCodeSummary struct {
	Attention int `json:"attention"`
	Running   int `json:"running"`
	Queued    int `json:"queued"`
}

func (a *desktopApp) startCodeStatusSync(ctx context.Context) {
	startDesktopStatusBar(a)
	wailsruntime.EventsOn(ctx, desktopCodeSummaryEvent, func(data ...interface{}) {
		if len(data) == 0 {
			return
		}
		summary, err := decodeDesktopCodeSummary(data[0])
		if err != nil {
			return
		}
		updateDesktopStatusBar(summary)
	})
}

func decodeDesktopCodeSummary(value interface{}) (desktopCodeSummary, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return desktopCodeSummary{}, err
	}
	var summary desktopCodeSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return desktopCodeSummary{}, err
	}
	summary.Attention = max(summary.Attention, 0)
	summary.Running = max(summary.Running, 0)
	summary.Queued = max(summary.Queued, 0)
	return summary, nil
}
