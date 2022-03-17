package schedulers

import (
	"fmt"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/schedulermain/apis"
	"log"
	"math/rand"
	"sync"
	"time"
)

const (
	HeartBeatTimeout = 5 * time.Second
)

type Schedulers struct {
	SchedulerMap           map[int]bool          // 保存scheduler-worker的map[schedulerID]bool
	SchedulerHeartBeatChan map[int]chan struct{} // schedulers-worker heart beat的chan
	SchedulerPartition     map[int][]string      // scheduler分区的node list
	NodeMap                map[string]bool       // 所有的node
	NodeLister             v1.NodeLister         // node lister
	SchedulerRWLock        sync.RWMutex          // 读写锁
}

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
	s.NewScheduler(schedulerID)

	// 返回schedulerID
	*reply = apis.RegisterReply{
		SchedulerID: schedulerID,
	}
	log.Printf("schedulers:%d have registered", schedulerID)

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

func (s *Schedulers) NewScheduler(schedulerID int) {
	s.SchedulerRWLock.Lock()
	s.SchedulerMap[schedulerID] = true
	s.SchedulerHeartBeatChan[schedulerID] = make(chan struct{})
	s.UpdateScheduler()
	s.SchedulerRWLock.Unlock()
}

func (s *Schedulers) DeleteScheduler(schedulerID int) {
	s.SchedulerRWLock.Lock()
	delete(s.SchedulerHeartBeatChan, schedulerID)
	delete(s.SchedulerMap, schedulerID)
	s.UpdateScheduler()
	log.Printf("schedulers:%d is offline", schedulerID)
	s.SchedulerRWLock.Unlock()
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

func NewSchedulers() Schedulers {
	return Schedulers{
		SchedulerMap:           make(map[int]bool),
		SchedulerHeartBeatChan: make(map[int]chan struct{}),
		SchedulerPartition:     make(map[int][]string),
		NodeMap:                make(map[string]bool),
		SchedulerRWLock:        sync.RWMutex{},
	}
}
