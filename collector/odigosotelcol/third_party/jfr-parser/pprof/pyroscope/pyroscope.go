package pyroscope

import (
	"strings"

	v1 "github.com/grafana/pyroscope/api/gen/proto/go/types/v1"
)

const (
	LabelNameDelta        = "__delta__"
	LabelNameProfileName  = "__name__"
	LabelNameServiceName  = "service_name"
	LabelNamePyroscopeSpy = "pyroscope_spy"
	LabelNameSessionID    = "__session_id__"
	LabelNameJfrEvent     = "jfr_event"
)

var allowedPrivateLabels = map[string]struct{}{
	LabelNameSessionID: {},
}

func IsLabelAllowedForIngestion(name string) bool {
	if !strings.HasPrefix(name, "__") {
		return true
	}
	_, allowed := allowedPrivateLabels[name]
	return allowed
}

func Labels(seriesLabels map[string]string, jfrEvent, metricName, appName, spyName string) []*v1.LabelPair {
	ls := make([]*v1.LabelPair, 0, len(seriesLabels)+5)
	for k, v := range seriesLabels {
		if !IsLabelAllowedForIngestion(k) {
			continue
		}
		ls = append(ls, &v1.LabelPair{
			Name:  k,
			Value: v,
		})
	}

	serviceNameLabelName := LabelNameServiceName
	for _, label := range ls {
		if label.Name == serviceNameLabelName {
			serviceNameLabelName = "app_name"
			break
		}
	}

	ls = append(ls,
		&v1.LabelPair{
			Name:  LabelNamePyroscopeSpy,
			Value: spyName,
		},
		&v1.LabelPair{
			Name:  LabelNameDelta,
			Value: "false",
		},
		&v1.LabelPair{
			Name:  LabelNameJfrEvent,
			Value: jfrEvent,
		},
		&v1.LabelPair{
			Name:  LabelNameProfileName,
			Value: metricName,
		},
	)
	if appName != "" {
		alreadySet := false
		for _, label := range ls {
			if label.Name == serviceNameLabelName {

				alreadySet = true
				break
			}
		}
		if !alreadySet {
			ls = append(ls, &v1.LabelPair{
				Name:  serviceNameLabelName,
				Value: appName,
			})
		}
	}
	return ls
}
