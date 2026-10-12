MDATAGENCMD ?= go run go.opentelemetry.io/collector/cmd/mdatagen@v0.159.0

.PHONY: mdatagen
mdatagen:
	GOWORK=off $(MDATAGENCMD) metadata.yaml
