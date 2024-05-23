package common

import (
	"encoding/json"
	"testing"
)

func TestMarshalOtelSdk(t *testing.T) {
	want := `{"sdkType":"native","sdkTier":"community"}`
	sdk := OtelSdkNativeCommunity
	got, err := json.Marshal(sdk)
	if err != nil {
		t.Errorf("Marshal error: %v", err)
	}
	if string(got) != want {
		t.Errorf("Marshal OtelSdk = %v, want %v", string(got), want)
	}
}

func TestUnmarshalOtelSdk(t *testing.T) {
	want := OtelSdkNativeCommunity
	data := []byte(`{"sdkType":"native","sdkTier":"community"}`)
	var got OtelSdk
	err := json.Unmarshal(data, &got)
	if err != nil {
		t.Errorf("Unmarshal error: %v", err)
	}
	if got != want {
		t.Errorf("Unmarshal OtelSdk = %v, want %v", got, want)
	}
}