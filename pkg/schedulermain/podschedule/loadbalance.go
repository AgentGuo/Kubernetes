package podschedule

import (
	"math/rand"
	"time"
)

// SchedulerLoadBalance 按照调度器权重抽取调度器，实现负载均衡
func (p *PodSchedule) SchedulerLoadBalance(podID int64) int {
	rand.Seed(time.Now().Unix() + podID)
	s := rand.Float64()
	var weightSum float64 = 0
	if len(p.SchedulerWeight) == 0 {
		panic("no scheduler to schedule pod")
		return 0
	}
	lastID := 0
	for id, weight := range p.SchedulerWeight {
		weightSum += weight
		if weightSum > s {
			return id
		}
		lastID = id
	}
	return lastID
}
