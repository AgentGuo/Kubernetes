package app

import (
	"io/ioutil"
	"sigs.k8s.io/yaml"
)

type SchedulerMainConf struct {
	SchedulerMainName string              `yaml:"schedulerMainName"`
	Port              int                 `yaml:"port"`
	Partition         map[string][]string `yaml:"partition"`
}

func GetConf(confPath string, conf *SchedulerMainConf) error {
	yamlFile, err := ioutil.ReadFile(confPath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlFile, conf)
	if err != nil {
		return err
	}
	return nil
}
