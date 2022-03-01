package schedulermain

import (
	"fmt"
	"k8s.io/kubernetes/pkg/schedulermain/apis"
	"log"
	"math/rand"
	"net"
	"net/rpc"
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

type SchedulerMain struct {
	SchedulerMap map[int]bool
	PodMap       map[int]*PodInfo
	M            sync.Mutex
}

type PodInfo struct {
	PodID       int
	SchedulerID int
	Status      int
	startTime   time.Time
}

func (s *SchedulerMain) RegisterScheduler(args apis.RegisterArgs, reply *apis.RegisterReply) error {
	var schedulerID int
	for {
		schedulerID = rand.Int()
		if _, ok := s.SchedulerMap[schedulerID]; !ok {
			break
		}
	}
	s.SchedulerMap[schedulerID] = true
	*reply = apis.RegisterReply{
		SchedulerID: schedulerID,
	}
	log.Printf("scheduler:%d have registered", schedulerID)
	return nil
}

func (s *SchedulerMain) RequestSchedule(args apis.RequestScheduleArgs, reply *apis.RequestScheduleReply) error {
	s.M.Lock()
	defer s.M.Unlock()
	if info, ok := s.PodMap[args.PodID]; ok { // pod在调度列表中
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
		s.PodMap[args.PodID] = &PodInfo{
			PodID:       args.PodID,
			SchedulerID: args.SchedulerID,
			Status:      PodScheduling,
			startTime:   time.Now(),
		}
		return permitSchedule(reply, true, nil)
	}
}

func (s *SchedulerMain) UpdatePodStatus(args apis.UpdatePodStatusArgs, reply *apis.UpdatePodStatusReply) error {
	s.M.Lock()
	defer s.M.Unlock()
	if info, ok := s.PodMap[args.PodID]; ok {
		(*info).Status = args.Status
		(*info).startTime = time.Now()
	}
	printPod(s.PodMap)
	return nil
}

func (s *SchedulerMain) DeletePod() {
	for {
		select {
		case <-time.After(3 * time.Second):
			s.M.Lock()
			for k, v := range s.PodMap {
				if v.Status == PodFinished && time.Now().Sub(v.startTime) > PodExpire { // 删除过期的调度完的pod
					delete(s.PodMap, k)
				}
			}
			s.M.Unlock()
		}
	}
}

func printPod(info map[int]*PodInfo) {
	if info == nil {
		return
	}
	for k, v := range info {
		log.Println("podID:", k, "status:", v.Status)
	}
}

func permitSchedule(reply *apis.RequestScheduleReply, b bool, err error) error {
	*reply = apis.RequestScheduleReply{
		IsPermitted: b,
	}
	return err
}

func (s *SchedulerMain) server(port int) {
	err := apis.RegisterService(s)
	if err != nil {
		log.Fatal("register error:", err)
		return
	}
	listener, err := net.Listen("tcp", ":"+fmt.Sprintf("%d", port))
	if err != nil {
		log.Fatal("listen error:", err)
		return
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatal("listen accept error:", err)
				continue
			}
			go rpc.ServeConn(conn)
		}
	}()
	go s.DeletePod()
}

func NewSchedulerMain() SchedulerMain {
	return SchedulerMain{
		SchedulerMap: make(map[int]bool),
		PodMap:       make(map[int]*PodInfo),
		M:            sync.Mutex{},
	}
}

func RunSchedulerMain(port int) {
	s := NewSchedulerMain()
	s.server(port)
	fmt.Println("[enter \"q\" or \"quit\" to stop]")
	input := ""
	for input != "q" && input != "quit" {
		fmt.Scanf("%s", &input)
	}
	fmt.Printf("%v\n", s.SchedulerMap)
}
