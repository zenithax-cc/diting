// cmd/client/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"hardware-collector/internal/collector"
	"hardware-collector/internal/config"
	"hardware-collector/internal/logger"
	"hardware-collector/internal/publisher"
)

func main() {
	configFile := flag.String("c", "/etc/hardware-collector/config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	log, err := logger.NewLogger(cfg.Logger.LogFile, cfg.Logger.Level, cfg.Logger.MaxSize, cfg.Logger.MaxBackups)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	// 限制资源使用
	runtime.GOMAXPROCS(cfg.Resource.CPUCores)

	// 初始化采集器
	coll, err := collector.NewCollector(cfg.Client.CacheDir)
	if err != nil {
		log.Fatalf("初始化采集器失败: %v", err)
	}

	// 初始化 Kafka 推送器
	pub := publisher.NewKafkaPublisher(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	defer pub.Close()

	// 启动采集任务
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(cfg.Client.Interval)
	defer ticker.Stop()

	// 监听信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Info("硬件采集客户端已启动")

	// 立即执行一次采集
	collectAndPublish(ctx, coll, pub, log)

	for {
		select {
		case <-ticker.C:
			collectAndPublish(ctx, coll, pub, log)
		case <-sigChan:
			log.Info("收到停止信号,正在退出...")
			return
		}
	}
}

func collectAndPublish(ctx context.Context, coll *collector.Collector, pub *publisher.KafkaPublisher, log *logger.Logger) {
	info, err := coll.Collect(ctx, nil)
	if err != nil {
		log.Errorf("采集失败: %v", err)
		return
	}

	if err := pub.Publish(ctx, info); err != nil {
		log.Errorf("推送失败: %v", err)
		return
	}

	log.Infof("成功采集并推送硬件信息")
}
