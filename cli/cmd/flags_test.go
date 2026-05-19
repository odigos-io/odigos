package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestMustGetBoolFlagUsesFlagValue(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "unset flag",
			args: nil,
			want: false,
		},
		{
			name: "implicit true flag",
			args: []string{"--yes"},
			want: true,
		},
		{
			name: "explicit false flag",
			args: []string{"--yes=false"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use: "test",
				Run: func(cmd *cobra.Command, args []string) {},
			}
			cmd.Flags().Bool("yes", false, "skip the confirmation prompt")
			cmd.SetArgs(tt.args)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute() returned error: %v", err)
			}

			if got := mustGetBoolFlag(cmd, "yes"); got != tt.want {
				t.Fatalf("mustGetBoolFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}
