package runner

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/vietrix/vbuild/internal/config"
)

type resourceManager struct {
	mu         sync.Mutex
	cpuAvail   int
	memAvail   int64
	gpuAvail   []string
	groupLimit map[string]int
	groupInUse map[string]int
	root       string
	log        *logger
}

type resourceLease struct {
	env     map[string]string
	release func()
}

func (r *Runner) acquireResources(ctx context.Context, task *config.Task, group string) (*resourceLease, error) {
	if r.resources == nil || task == nil {
		return nil, nil
	}
	req := task.Resources
	if req == nil && group != "" {
		req = &config.ResourceRequest{}
	}
	return r.resources.Acquire(ctx, req, group)
}

func newResourceManager(pool *config.ResourcePool, root string, log *logger) *resourceManager {
	if pool == nil {
		return nil
	}
	cpu := pool.CPU
	if cpu == 0 {
		cpu = runtime.NumCPU()
	}
	mem := int64(0)
	if pool.Memory != "" {
		if parsed, err := parseByteSize(pool.Memory); err == nil {
			mem = parsed
		} else if log != nil {
			log.Errorf("resources: %v\n", err)
		}
	}
	gpuAvail := []string{}
	if len(pool.GPUDevices) > 0 {
		gpuAvail = append([]string(nil), pool.GPUDevices...)
	} else if pool.GPUs > 0 {
		for i := 0; i < pool.GPUs; i++ {
			gpuAvail = append(gpuAvail, fmt.Sprintf("%d", i))
		}
	}
	groupLimit := map[string]int{}
	for k, v := range pool.Groups {
		if v > 0 {
			groupLimit[k] = v
		}
	}
	return &resourceManager{
		cpuAvail:   cpu,
		memAvail:   mem,
		gpuAvail:   gpuAvail,
		groupLimit: groupLimit,
		groupInUse: map[string]int{},
		root:       root,
		log:        log,
	}
}

func (m *resourceManager) Acquire(ctx context.Context, req *config.ResourceRequest, group string) (*resourceLease, error) {
	if m == nil || req == nil {
		return nil, nil
	}
	cpu := req.CPU
	mem := int64(0)
	if req.Memory != "" {
		parsed, err := parseByteSize(req.Memory)
		if err != nil {
			return nil, err
		}
		mem = parsed
	}
	gpu := req.GPU
	if cpu <= 0 && mem <= 0 && gpu <= 0 && group == "" {
		return nil, nil
	}
	wait := 100 * time.Millisecond
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if devices, ok := m.acquire(cpu, mem, gpu, group); ok {
			env := map[string]string{}
			if len(devices) > 0 {
				list := strings.Join(devices, ",")
				env["CUDA_VISIBLE_DEVICES"] = list
				env["GPU_VISIBLE_DEVICES"] = list
				env["ROCR_VISIBLE_DEVICES"] = list
				env["VBUILD_GPU_DEVICES"] = list
				env["VBUILD_GPU_COUNT"] = fmt.Sprintf("%d", len(devices))
			}
			if cpu > 0 {
				env["VBUILD_CPU_REQUEST"] = fmt.Sprintf("%d", cpu)
			}
			if mem > 0 {
				env["VBUILD_MEMORY_REQUEST"] = fmt.Sprintf("%d", mem)
			}
			release := func() {
				m.release(cpu, mem, devices, group)
			}
			return &resourceLease{env: env, release: release}, nil
		}
		time.Sleep(wait)
		if wait < 500*time.Millisecond {
			wait += 50 * time.Millisecond
		}
	}
}

func (m *resourceManager) acquire(cpu int, mem int64, gpu int, group string) ([]string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cpu > 0 && m.cpuAvail < cpu {
		return nil, false
	}
	if mem > 0 && m.memAvail > 0 && m.memAvail < mem {
		return nil, false
	}
	if gpu > 0 && len(m.gpuAvail) < gpu {
		return nil, false
	}
	if group != "" {
		limit := m.groupLimit[group]
		if limit > 0 && m.groupInUse[group] >= limit {
			return nil, false
		}
	}
	assigned := []string{}
	if gpu > 0 {
		assigned = append([]string(nil), m.gpuAvail[:gpu]...)
		m.gpuAvail = append([]string(nil), m.gpuAvail[gpu:]...)
	}
	if cpu > 0 {
		m.cpuAvail -= cpu
	}
	if mem > 0 && m.memAvail > 0 {
		m.memAvail -= mem
	}
	if group != "" {
		m.groupInUse[group]++
	}
	return assigned, true
}

func (m *resourceManager) release(cpu int, mem int64, devices []string, group string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cpu > 0 {
		m.cpuAvail += cpu
	}
	if mem > 0 && m.memAvail > 0 {
		m.memAvail += mem
	}
	if len(devices) > 0 {
		m.gpuAvail = append(m.gpuAvail, devices...)
	}
	if group != "" {
		if m.groupInUse[group] > 0 {
			m.groupInUse[group]--
		}
	}
}
