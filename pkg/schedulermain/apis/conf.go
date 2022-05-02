package apis

import (
	"io/ioutil"
	"sigs.k8s.io/yaml"
)

// SchedulerMainConf scheduler main的配置文件结构体
type SchedulerMainConf struct {
	SchedulerMainName       string              `yaml:"schedulerMainName"`       // 调度器名称
	Port                    int                 `yaml:"port"`                    // 运行端口，默认为12345
	Partition               map[string][]string `yaml:"partition"`               // 分区信息，如果不知道则不固定分区
	V                       int                 `yaml:"V"`                       // 日志详细程度级别
	ScheduleTimeoutInterval int                 `yaml:"scheduleTimeoutInterval"` // 调度超时时间
	PodExpire               int                 `yaml:"podExpire"`               // 已调度pod垃圾回收时间
	HeartBeatTimeout        int                 `yaml:"heartBeatTimeout"`        // scheduler worker心跳超时时间
}

// GetConf 从文件中解析得到配置信息
func GetConf(confPath string, conf *SchedulerMainConf) error {
	yamlFile, err := ioutil.ReadFile(confPath)
	if err != nil {
		return err
	}
	// yaml反序列化
	err = yaml.Unmarshal(yamlFile, conf)
	if err != nil {
		return err
	}
	return nil
}
