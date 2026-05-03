package api

import (
	"net/http"
	"time"
)

var aiPreviewProbeClient = &http.Client{Timeout: 2 * time.Second}
