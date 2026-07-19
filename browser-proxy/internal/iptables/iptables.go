// Package iptables applies the transparent inbound traffic redirect for the odigos-browser-proxy
// sidecar, modeled on the Istio init-container approach: inbound TCP destined to the application
// port is redirected to the sidecar's listen port, while the sidecar's own traffic (matched by UID)
// is excluded so it can reach the application on loopback without looping.
package iptables

import (
	"fmt"
	"os/exec"
	"strconv"
)

// Config parameters for the redirect.
type Config struct {
	AppPort   int
	ProxyPort int
	ProxyUID  int
}

// Apply installs the nat rules. It requires CAP_NET_ADMIN (the init container runs with that
// capability). The `iptables` binary must be present in the image.
func Apply(cfg Config) error {
	appPort := strconv.Itoa(cfg.AppPort)
	proxyPort := strconv.Itoa(cfg.ProxyPort)
	uid := strconv.Itoa(cfg.ProxyUID)

	rules := [][]string{
		// Inbound (from outside the pod): redirect TCP to the application port -> sidecar port.
		{"-t", "nat", "-A", "PREROUTING", "-p", "tcp", "--dport", appPort, "-j", "REDIRECT", "--to-ports", proxyPort},

		// Locally generated traffic owned by the sidecar UID must NOT be redirected, so the sidecar
		// can connect to the application on 127.0.0.1:<appPort>.
		{"-t", "nat", "-A", "OUTPUT", "-p", "tcp", "--dport", appPort, "-m", "owner", "--uid-owner", uid, "-j", "RETURN"},
		// Any other local process targeting the application port is redirected to the sidecar too,
		// so in-pod clients are also instrumented consistently.
		{"-t", "nat", "-A", "OUTPUT", "-p", "tcp", "--dport", appPort, "-j", "REDIRECT", "--to-ports", proxyPort},
	}

	for _, rule := range rules {
		// "-w" makes iptables wait for the xtables lock instead of failing if it is held.
		args := append([]string{"-w"}, rule...)
		cmd := exec.Command("iptables", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("iptables %v failed: %w: %s", rule, err, string(out))
		}
	}

	return nil
}
