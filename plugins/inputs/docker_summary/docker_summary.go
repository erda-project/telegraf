package dockersummary

import (
	"fmt"
	"log"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/filter"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/inputs/global/kubelet"
	"github.com/influxdata/telegraf/plugins/inputs/global/node"
)

// Summary .
type Summary struct {
	Enable          bool     `toml:"enable"`
	EnvInclude      []string `toml:"env_include"`
	LabelInclude    []string `toml:"label_include"`
	HostMountPrefix string   `toml:"host_mount_prefix"`

	labelFilter filter.Filter
	envFilter   filter.Filter
	initialize  bool

	k8s          bool
	podNetStatus map[kubelet.PodID]*kubelet.PodStatus
	// pods         map[kubernetes.PodId]*apiv1.Pod
}

// Description .
func (*Summary) Description() string { return "" }

// SampleConfig .
func (*Summary) SampleConfig() string { return "" }

// Gather .
// 日志等级调整为警告，因为后续高版本kubernetes(v1.22+)可能不会依赖docker采集容器指标
func (s *Summary) Gather(acc telegraf.Accumulator) (err error) {
	if !s.Enable {
		return nil
	}
	if !s.initialize {
		err := s.init(acc)
		if err != nil {
			log.Printf("W! [docker_summary] init err: %s", err)
			return nil
		}
	}

	// List containers
	filterArgs := filters.NewArgs()
	filterArgs.Add("status", "running")
	containers, err := s.dockerList(types.ContainerListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		log.Printf("W! [docker_summary] list containers err: %s", err)
		return nil
	}

	if s.k8s {
		err := s.getKubernetesInfo()
		if err != nil {
			log.Printf("W! fail to get kubernetes info: %s", err)
		}
	}

	// Get container data
	var wg sync.WaitGroup
	wg.Add(len(containers))
	for _, container := range containers {
		go func(c types.Container) {
			defer wg.Done()
			err := s.gatherContainer(c.ID, &c, acc)
			if err != nil {
				log.Printf("W! Error gathering container %v stats: %s", c.Names, err)
			}
		}(container)
	}
	wg.Wait()

	if s.k8s {
		s.podNetStatus = nil
	}
	return nil
}

func (s *Summary) getKubernetesInfo() error {
	summary, err := kubelet.GetStatsSummery()
	if err != nil {
		s.podNetStatus = nil
		return fmt.Errorf("fail to kubelet.GetStatsSummery: %s", err)
	} else {
		s.podNetStatus = summary
	}
	return nil
}

func (s *Summary) init(acc telegraf.Accumulator) error {
	if s.initialize {
		return nil
	}
	info := node.GetInfo()
	if info != nil && info.IsK8s() {
		s.k8s = true
	}

	envFilter, err := filter.Compile(s.EnvInclude)
	if err != nil {
		return err
	}
	s.envFilter = envFilter
	labelFilter, err := filter.Compile(s.LabelInclude)
	if err != nil {
		return err
	}
	s.labelFilter = labelFilter

	err = s.startDockerEventLoop(acc)
	if err != nil {
		log.Printf("fail to start docker event loop: %s", err)
		return err
	}
	s.initialize = true
	return nil
}

// Start .
func (s *Summary) Start(acc telegraf.Accumulator) error { return nil }

// Stop .
func (s *Summary) Stop() {}

func init() {
	inputs.Add("docker_summary", func() telegraf.Input {
		return &Summary{}
	})
}
