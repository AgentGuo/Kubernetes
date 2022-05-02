package podschedule

import (
	"fmt"
	"k8s.io/kubernetes/pkg/schedulermain/apis"
	"k8s.io/kubernetes/pkg/schedulermain/utils"
	"sync"
	"time"
)

// pod调度状态常量
const (
	PodScheduling = 1
	PodFailed     = 2
	PodFinished   = 3
)

var (
	ScheduleTimeoutInterval = 5 * time.Second
	PodExpire               = 3 * time.Second
)

type PodSchedule struct {
	PodMap          map[int]*PodInfo // 保存pod的map
	SchedulerWeight map[int]float64  // scheduler调度的权重
	PodRWLock       sync.RWMutex     // pod读写锁
}

type PodInfo struct {
	PodID       int       // podID，由podName和podNamespace通过md5计算得到
	SchedulerID int       // 分配的scheduler worker ID
	Status      int       // pod状态
	startTime   time.Time // pod上次调度时间
}

func NewPodSchedule() PodSchedule {
	return PodSchedule{
		PodMap:          make(map[int]*PodInfo),
		SchedulerWeight: make(map[int]float64),
		PodRWLock:       sync.RWMutex{},
	}
}

// RequestSchedule 请求调度pod
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
		utils.LogV(2, fmt.Sprintf("[pod scheduling] pod-%d assigned to scheduler-%d",
			args.PodID, podScheduleID))
		if podScheduleID == args.SchedulerID {
			return permitSchedule(reply, true, nil)
		} else {
			return permitSchedule(reply, false, nil)
		}
	}
}

// permitSchedule 允许调度
func permitSchedule(reply *apis.RequestScheduleReply, b bool, err error) error {
	*reply = apis.RequestScheduleReply{
		IsPermitted: b,
	}
	return err
}

// UpdatePodStatus 更新调度状态
func (p *PodSchedule) UpdatePodStatus(args apis.UpdatePodStatusArgs, reply *apis.UpdatePodStatusReply) error {
	p.PodRWLock.Lock()
	defer p.PodRWLock.Unlock()
	// 更新调度状态并写入日志
	if info, ok := p.PodMap[args.PodID]; ok {
		(*info).Status = args.Status
		(*info).startTime = time.Now()
		if args.Status == PodFinished {
			utils.LogV(2, fmt.Sprintf("[pod finished] pod-%d schedule finished",
				args.PodID))
		} else {
			utils.LogV(2, fmt.Sprintf("[pod failed] pod-%d schedule failed",
				args.PodID))
		}
	}
	return nil
}

// DeletePod 后台删除pod缓存
func (p *PodSchedule) DeletePod() {
	for {
		select {
		// 每3秒检查清空一次
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
