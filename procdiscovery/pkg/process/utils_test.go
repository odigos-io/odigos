package process

import "testing"

func TestIsDirectoryProcess_IsNumeric(t *testing.T) {
	pid, isProcess := isDirectoryPid("123")
	if !isProcess {
		t.Errorf("isDirectoryProcess() = %v, want %v", isProcess, true)
	}
	if pid != 123 {
		t.Errorf("isDirectoryProcess() = %v, want %v", pid, 123)
	}
}

func TestIsDirectoryProcess_IsNotNumeric(t *testing.T) {
	pid, isProcess := isDirectoryPid("abc")
	if isProcess {
		t.Errorf("isDirectoryProcess() = %v, want %v", isProcess, false)
	}
	if pid != 0 {
		t.Errorf("isDirectoryProcess() = %v, want %v", pid, 0)
	}
}
