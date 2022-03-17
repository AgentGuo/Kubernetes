package podschedule

import (
	"k8s.io/kubernetes/pkg/schedulermain/apis"
	"log"
	"sync"
	"time"
)

const (
	PodScheduling = 1
	PodFailed     = 2
	PodFinished   = 3
)

const (
	ScheduleTimeoutInterval = 5 * time.Second
	PodExpire               = 3 * time.Second
)

type PodSchedule struct {
	PodMap          map[int]*PodInfo // 保存pod的map
	SchedulerWeight map[int]float64  // scheduler调度的权重
	PodRWLock       sync.RWMutex     // pod读写锁
}

type PodInfo struct {
	PodID       int
	SchedulerID int
	Status      int
	startTime   time.Time
}

func NewPodSchedule() PodSchedule {
	return PodSchedule{
		PodMap:          make(map[int]*PodInfo),
		SchedulerWeight: make(map[int]float64),
		PodRWLock:       sync.RWMutex{},
	}
}

func (p *PodSchedule) RequestSchedule(args apis.RequestScheduleArgs, reply *apis.RequestScheduleReply) error {
	p.PodRWLock.Lock()
	defer p.PodRWLock.Unlock()
	if info, ok := p.PodMap[args.PodID]; ok { // pod在调度列表中
		if info.SchedulerID == args.SchedulerID { // 已分配，直接返回
			return permitSchedule(reply, true, nil)
		}
		if info.Status == PodScheduling { // Pod正在被调度
			if time.Now().Sub(info.startTime) > ScheduleTimeoutInterval { // 超时 重新调度
				return permitSchedule(reply, true, nil)
			} else { // 未超时
				return permitSchedule(reply, false, nil)
			}
		} else if info.Status == PodFinished { // pod调度完成
			return permitSchedule(reply, false, nil)
		} else if info.Status == PodFailed { // pod调度失败
			return permitSchedule(reply, true, nil)
		} else {
			return permitSchedule(reply, false, nil)
		}
	} else { // pod不在调度列表中
		podScheduleID := p.SchedulerLoadBalance(int64(args.PodID))
		p.PodMap[args.PodID] = &PodInfo{
			PodID:       args.PodID,
			SchedulerID: podScheduleID,
			Status:      PodScheduling,
			startTime:   time.Now(),
		}
		if podScheduleID == args.SchedulerID {
			return permitSchedule(reply, true, nil)
		} else {
			return permitSchedule(reply, false, nil)
		}
	}
}

func permitSchedule(reply *apis.RequestScheduleReply, b bool, err error) error {
	*reply = apis.RequestScheduleReply{
		IsPermitted: b,
	}
	return err
}

func (p *PodSchedule) UpdatePodStatus(args apis.UpdatePodStatusArgs, reply *apis.UpdatePodStatusReply) error {
	p.PodRWLock.Lock()
	defer p.PodRWLock.Unlock()
	if info, ok := p.PodMap[args.PodID]; ok {
		(*info).Status = args.Status
		(*info).startTime = time.Now()
	}
	printPod(p.PodMap)
	return nil
}

func printPod(info map[int]*PodInfo) {
	if info == nil {
		return
	}
	for k, v := range info {
		log.Println("podID:", k, "status:", v.Status)
	}
}

func (p *PodSchedule) DeletePod() {
	for {
		select {
		case <-time.After(3 * time.Second):
			p.PodRWLock.Lock()
			for k, v := range p.PodMap {
				if v.Status == PodFinished && time.Now().Sub(v.startTime) > PodExpire { // 删除过期的调度完的pod
					delete(p.PodMap, k)
				}
			}
			p.PodRWLock.Unlock()
		}
	}
}
