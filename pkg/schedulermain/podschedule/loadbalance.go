package podschedule

import (
	"math/rand"
	"time"
)

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
