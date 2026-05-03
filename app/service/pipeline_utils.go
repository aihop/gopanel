package service

import "strings"

var archiveExcludedNames = map[string]struct{}{".git": {}, ".gopanel_artifact": {}, "node_modules": {}, "__MACOSX": {}}
var releaseExcludedNames = map[string]struct{}{".git": {}, ".gopanel_artifact": {}, ".gopanel_shims": {}, "node_modules": {}, "__MACOSX": {}}

type logWriter struct {
	logger *PipelineLogger
	isErr  bool
}

func newLogWriter(logger *PipelineLogger, isErr bool) *logWriter {
	return &logWriter{logger: logger, isErr: isErr}
}
func (w *logWriter) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if w.isErr {
			w.logger.Error("%s", line)
		} else {
			w.logger.Info("%s", line)
		}
	}
	return len(p), nil
}
