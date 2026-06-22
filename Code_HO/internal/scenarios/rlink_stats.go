package scenarios

import (
	"fmt"
	"stormsim/internal/core/gnbcontext"
	"stormsim/internal/core/uecontext"
	"stormsim/internal/transport/rlink"
)

func printMeasurementRlinkStats(ueCtx *uecontext.UeContext, gnbs map[string]*gnbcontext.GnbContext) {
	fmt.Println("\n========== RLink Stats ==========")

	ueStats := ueCtx.GetRlinkStats()
	fmt.Printf("  UE [msin=%s]:  Sent=%d  Delivered=%d  Dropped=%d  Timeout=%d\n",
		ueCtx.GetMsin(), ueStats.Sent, ueStats.Delivered, ueStats.Dropped, ueStats.Timeout)

	var total rlink.Stats
	for gnbId, gnb := range gnbs {
		snapshot := gnb.GetRlinkStatsSnapshot()
		if len(snapshot) == 0 {
			continue
		}
		fmt.Printf("  gNB [%s]: %d connection(s)\n", gnbId, len(snapshot))
		for key, stats := range snapshot {
			fmt.Printf("    %s:  Sent=%d  Delivered=%d  Dropped=%d  Timeout=%d\n",
				key, stats.Sent, stats.Delivered, stats.Dropped, stats.Timeout)
			total.Sent += stats.Sent
			total.Delivered += stats.Delivered
			total.Dropped += stats.Dropped
			total.Timeout += stats.Timeout
		}
	}

	total.Sent += ueStats.Sent
	total.Delivered += ueStats.Delivered
	total.Dropped += ueStats.Dropped
	total.Timeout += ueStats.Timeout

	fmt.Printf("  TOTAL:  Sent=%d  Delivered=%d  Dropped=%d  Timeout=%d\n",
		total.Sent, total.Delivered, total.Dropped, total.Timeout)
	fmt.Println("=================================")
}
