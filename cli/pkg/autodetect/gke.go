package autodetect

type gkeDetector struct{}

func (g gkeDetector) Detect(args DetectionArguments) (Kind, error) {
	return KindGKE, nil
}
