package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/aihop/gopanel/gpc/internal/monitor"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:          "status",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")

		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout())
		defer cancel()

		st, err := monitor.Collect(ctx)
		if err != nil {
			return err
		}

		if jsonOut {
			b, err := json.Marshal(st)
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, string(b))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintf(w, "At\t%s\n", time.UnixMilli(st.AtUnixMs).Format(time.RFC3339))
		fmt.Fprintf(w, "CPU\t%.2f%%\n", st.CPUPercent)
		fmt.Fprintf(w, "Mem\t%.2f%%\t(%s/%s)\n", st.MemUsedPercent, bytesHuman(st.MemUsed), bytesHuman(st.MemTotal))
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "Mount\tFS\tUsed%\tUsed\tTotal\tFree")
		for _, d := range st.Disks {
			fmt.Fprintf(w, "%s\t%s\t%.2f%%\t%s\t%s\t%s\n", d.Mountpoint, d.Fstype, d.UsedPercent, bytesHuman(d.Used), bytesHuman(d.Total), bytesHuman(d.Free))
		}
		return w.Flush()
	},
}

func init() {
	statusCmd.Flags().Bool("json", false, "output json")
}

func bytesHuman(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := uint64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

