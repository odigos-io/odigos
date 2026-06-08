package distros

import (
	"testing"
)

func TestNodejsCommunityLoadsOwnLogsSupport(t *testing.T) {
	getter, err := NewCommunityGetter()
	if err != nil {
		t.Fatalf("NewCommunityGetter() returned error: %v", err)
	}

	nodejs := getter.GetDistroByName("nodejs-community")
	if nodejs == nil {
		t.Fatal("nodejs-community distro was not loaded")
	}
	if nodejs.OwnLogs == nil {
		t.Fatal("nodejs-community distro did not load ownLogs support")
	}
	if !nodejs.OwnLogs.OdigosAgentOwnLogerSupported {
		t.Fatal("nodejs-community distro should support Odigos agent own logs")
	}
	if !nodejs.OwnLogs.OpenTelemetryComponentsLoggerSupported {
		t.Fatal("nodejs-community distro should support OpenTelemetry components logs")
	}
}
