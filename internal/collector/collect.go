package collector

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"hardware-collector/pkg/models"
)

type Collector struct {
	cache    *Cache
	mu       sync.RWMutex
	lastData *models.HardwareInfo
}

func NewCollector(cacheDir string) (*Collector, error) {
	cache, err := NewCache(cacheDir)
	if err != nil {
		return nil, err
	}

	return &Collector{
		cache: cache,
	}, nil
}

func (c *Collector) Collect(ctx context.Context, modules []string) (*models.HardwareInfo, error) {
	info := &models.HardwareInfo{
		Timestamp: time.Now(),
	}

	hostname, _ := os.Hostname()
	info.Hostname = hostname

	// 根据指定模块采集信息
	moduleSet := make(map[string]bool)
	if len(modules) == 0 {
		moduleSet = map[string]bool{
			"system": true, "memory": true, "disk": true,
			"network": true, "gpu": true,
		}
	} else {
		for _, m := range modules {
			moduleSet[m] = true
		}
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 5)

	if moduleSet["system"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if sysInfo, err := c.collectSystemInfo(ctx); err != nil {
				errChan <- err
			} else {
				info.System = sysInfo
			}
		}()
	}

	if moduleSet["memory"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if memInfo, err := c.collectMemoryInfo(ctx); err != nil {
				errChan <- err
			} else {
				info.Memory = memInfo
			}
		}()
	}

	if moduleSet["disk"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if diskInfo, err := c.collectDiskInfo(ctx); err != nil {
				errChan <- err
			} else {
				info.Disk = diskInfo
			}
		}()
	}

	if moduleSet["network"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if netInfo, err := c.collectNetworkInfo(ctx); err != nil {
				errChan <- err
			} else {
				info.Network = netInfo
			}
		}()
	}

	if moduleSet["gpu"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if gpuInfo, err := c.collectGPUInfo(ctx); err != nil {
				// GPU 采集失败不影响其他模块
				errChan <- nil
			} else {
				info.GPU = gpuInfo
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// 检查是否有错误
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	// 检查缓存,判断是否需要更新
	if c.shouldUpdate(info) {
		c.updateCache(info)
	}

	return info, nil
}

func (c *Collector) shouldUpdate(newInfo *models.HardwareInfo) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastData == nil {
		return true
	}

	// 比较数据是否有变化
	oldJSON, _ := json.Marshal(c.lastData)
	newJSON, _ := json.Marshal(newInfo)

	return string(oldJSON) != string(newJSON)
}

func (c *Collector) updateCache(info *models.HardwareInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastData = info
	_ = c.cache.Save(info)
}
