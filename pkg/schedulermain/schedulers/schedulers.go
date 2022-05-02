package schedulers

import (
	"fmt"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/schedulermain/apis"
	"k8s.io/kubernetes/pkg/schedulermain/utils"
	"math/rand"
	"sync"
	"time"
)

var (
	HeartBeatTimeout = 5 * time.Second
)

type Schedulers struct {
	SchedulerMap           map[int]bool          // 保存scheduler-worker的map[schedulerID]bool
	SchedulerHeartBeatChan map[int]chan struct{} // schedulers-worker heart beat的chan
	SchedulerNodeList      map[int][]string      // scheduler-worker分区的node list
	NodeMap                map[string]bool       // 所有的node
	NodePartition          map[string][]string   // node分区（只有在手动分区时才使用）
	PartitionScheduler     map[string]int        // 分区到scheduler-worker的影射（只有在手动分区时才使用）
	NodeLister             v1.NodeLister         // node lister
	SchedulerRWLock        sync.RWMutex          // 读写锁
}

// RegisterScheduler 注册scheduler
func (s *Schedulers) RegisterScheduler(args apis.RegisterArgs, reply *apis.RegisterReply) error {
	var schedulerID int
	// 随机分配schedulerID
	for {
		schedulerID = rand.Int()
		if _, ok := s.SchedulerMap[schedulerID]; !ok {
			break
		}
	}
	// 添加scheduler
	partition, err := s.NewScheduler(schedulerID)
	if err != nil {
		return err
	}

	// 返回schedulerID
	*reply = apis.RegisterReply{
		SchedulerID:       schedulerID,
		SchedulePartition: partition,
	}

	// 为每个scheduler运行一个heart beat协程检测
	go func() {
		for {
			select {
			case <-time.After(HeartBeatTimeout): // 超时，认为scheduler离线，删除scheduler
				s.DeleteScheduler(schedulerID)
				return
			case <-s.SchedulerHeartBeatChan[schedulerID]: // 正常heart beat
				continue
			}
		}
	}()
	return nil
}

func (s *Schedulers) NewScheduler(schedulerID int) (string, error) {
	partition := ""
	s.SchedulerRWLock.Lock()
	defer s.SchedulerRWLock.Unlock()
	if s.NodePartition != nil && len(s.NodePartition) != 0 {
		if len(s.PartitionScheduler) >= len(s.NodePartition) {
			return partition, fmt.Errorf("no partitions left")
		}
		for k, _ := range s.NodePartition {
			if _, ok := s.PartitionScheduler[k]; !ok {
				partition = k
				s.PartitionScheduler[k] = schedulerID
				break
			}
		}
	}
	s.SchedulerMap[schedulerID] = true
	s.SchedulerHeartBeatChan[schedulerID] = make(chan struct{})
	s.UpdateSchedulerPartition()
	utils.LogV(0, fmt.Sprintf("[scheduler add] scheduler-%d is added", schedulerID))
	return partition, nil
}

func (s *Schedulers) DeleteScheduler(schedulerID int) {
	s.SchedulerRWLock.Lock()
	defer s.SchedulerRWLock.Unlock()
	if s.NodePartition != nil && len(s.NodePartition) != 0 {
		for k, v := range s.PartitionScheduler {
			if v == schedulerID {
				delete(s.PartitionScheduler, k)
				break
			}
		}
	}
	delete(s.SchedulerHeartBeatChan, schedulerID)
	delete(s.SchedulerMap, schedulerID)
	s.UpdateSchedulerPartition()
	utils.LogV(0, fmt.Sprintf("[scheduler delete] schedulers-%d is offline", schedulerID))
}

func (s *Schedulers) HeartBeat(args apis.HeartBeatArgs, reply *apis.HeartBeatReply) error {
	if _, ok := s.SchedulerHeartBeatChan[args.SchedulerID]; ok {
		s.SchedulerRWLock.Lock()
		s.SchedulerHeartBeatChan[args.SchedulerID] <- struct{}{}
		s.SchedulerRWLock.Unlock()
		return nil
	} else {
		return fmt.Errorf("schedulers%d is offline", args.SchedulerID)
	}
}

func NewSchedulers(partition map[string][]string) Schedulers {
	return Schedulers{
		SchedulerMap:           make(map[int]bool),
		SchedulerHeartBeatChan: make(map[int]chan struct{}),
		SchedulerNodeList:      make(map[int][]string),
		NodeMap:                make(map[string]bool),
		NodePartition:          partition,
		PartitionScheduler:     make(map[string]int),
		SchedulerRWLock:        sync.RWMutex{},
	}
}
