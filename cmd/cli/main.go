// cmd/cli/main.go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"hardware-collector/internal/collector"
	"hardware-collector/pkg/models"
)

func main() {
	modules := flag.String("m", "", "采集模块(system,memory,disk,network,gpu),逗号分隔")
	detailed := flag.Bool("d", false, "显示详细信息")
	jsonOutput := flag.Bool("j", false, "JSON格式输出")
	debug := flag.Bool("D", false, "调试模式")
	flag.Parse()

	coll, err := collector.NewCollector("/tmp/hardware-collector")
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		os.Exit(1)
	}

	var moduleList []string
	if *modules != "" {
		moduleList = strings.Split(*modules, ",")
	}

	ctx := context.Background()
	info, err := coll.Collect(ctx, moduleList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "采集失败: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		data, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(data))
	} else if *detailed {
		printDetailed(info)
	} else {
		printSimple(info)
	}

	if *debug {
		fmt.Printf("\n[DEBUG] 采集时间: %s\n", info.Timestamp)
	}
}

func printSimple(info *models.HardwareInfo) {
	fmt.Printf("主机名: %s\n", info.Hostname)
	if info.System != nil {
		fmt.Printf("系统: %s %s\n", info.System.OS, info.System.PlatformVersion)
		fmt.Printf("CPU: %s (%d核/%d线程)\n", info.System.CPUModel, info.System.CPUCores, info.System.CPUThreads)
	}
	if info.Memory != nil {
		fmt.Printf("内存: %.2fGB / %.2fGB (%.1f%%)\n",
			float64(info.Memory.Used)/1024/1024/1024,
			float64(info.Memory.Total)/1024/1024/1024,
			info.Memory.UsedPercent)
	}
	if len(info.Disk) > 0 {
		fmt.Printf("磁盘: %d个分区\n", len(info.Disk))
	}
	if len(info.Network) > 0 {
		fmt.Printf("网络: %d个接口\n", len(info.Network))
	}
	if len(info.GPU) > 0 {
		fmt.Printf("GPU: %d个\n", len(info.GPU))
	}
}

func printDetailed(info *models.HardwareInfo) {
	data, _ := json.MarshalIndent(info, "", "  ")
	fmt.Println(string(data))
}
