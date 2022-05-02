package app

import (
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/schedulermain"
	"k8s.io/kubernetes/pkg/schedulermain/apis"
	"log"
	"os"
	"sigs.k8s.io/yaml"
)

var (
	conf = &apis.SchedulerMainConf{
		SchedulerMainName:       "schedulermain",
		Port:                    12345,
		Partition:               nil,
		V:                       0,
		ScheduleTimeoutInterval: 5,
		PodExpire:               3,
		HeartBeatTimeout:        5,
	}
	cfgFile string
	rootCmd = cobra.Command{
		Use:   "kube-schedulers-main",
		Short: "A brief description of your application",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(cfgFile) != 0 {
				err := apis.GetConf(cfgFile, conf)
				if err != nil {
					return
				}
			}
			//fmt.Printf("%v\n", conf)
			confYaml, _ := yaml.Marshal(conf)
			log.Printf(`[schedule-main started] sever on port %d
----------config info----------
%s
-------------------------------`, conf.Port, string(confYaml))
			schedulermain.RunSchedulerMain(conf)
		},
	}
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobralearn.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.Flags().IntVarP(&(conf.Port), "port", "p", 12345, "schedulers-main service port")
	rootCmd.Flags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/schedulermain.yaml)")
}
