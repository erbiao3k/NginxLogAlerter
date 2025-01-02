package main

import (
	"NginxLogAlerter/config"
	"NginxLogAlerter/public"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/allegro/bigcache/v3"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// NginxLog 定义了 Nginx 日志的结构
type NginxLog struct {
	Timestamp time.Time `json:"@timestamp"`
	Metadata  struct {
		Beat    string `json:"beat"`
		Type    string `json:"type"`
		Version string `json:"version"`
		Topic   string `json:"topic"`
	} `json:"@metadata"`
	Source  string `json:"source"`
	Offset  int64  `json:"offset"`
	Message string `json:"message"`
	Fields  struct {
		KafkaTopic string `json:"kafka_topic"`
	} `json:"fields"`
	Beat struct {
		Name     string `json:"name"`
		Hostname string `json:"hostname"`
		Version  string `json:"version"`
	} `json:"beat"`
	Host struct {
		Name string `json:"name"`
	} `json:"host"`
}

func main() {
	// 初始化内存缓存
	cache, err := bigcache.New(context.Background(), bigcache.Config{
		Shards:       2,
		LifeWindow:   10 * time.Minute,
		CleanWindow:  1 * time.Second,
		MaxEntrySize: 500, // 限制缓存项的最大大小
	})
	if err != nil {
		log.Fatalf("初始化内存缓存失败：%v", err)
	}
	defer cache.Close()

	// 设置 Kafka 日志输出
	sarama.Logger = log.New(os.Stdout, "[Kafka框架日志] ", log.LstdFlags)

	// Kafka 消费者配置
	cfg := sarama.NewConfig()
	cfg.ClientID = "go-dayu-NginxLogAlerter-kafka-client"
	cfg.Consumer.Return.Errors = true
	cfg.Version = sarama.V0_10_1_1

	// 创建 Kafka 消费者
	consumer, err := sarama.NewConsumer(config.Brokers, cfg)
	if err != nil {
		log.Fatalf("启动消费端失败: %v", err)
	}
	defer consumer.Close()

	// 获取所有分区
	partitionList, err := consumer.Partitions(config.Topic)
	if err != nil {
		log.Fatalf("根据topic取到所有的分区失败：%v", err)
	}

	// 并发消费每个分区的消息
	for partition := range partitionList {
		// 针对每个分区创建一个对应的分区消费者
		pc, err := consumer.ConsumePartition(config.Topic, int32(partition), sarama.OffsetNewest)
		if err != nil {
			log.Printf("failed to start consumer for partition %d, err：%v\n", partition, err)
			continue
		}
		defer pc.AsyncClose()

		// 异步从每个分区消费信息
		go consumePartition(pc, cache)
	}
}

// consumePartition 消费指定分区的消息
func consumePartition(pc sarama.PartitionConsumer, cache *bigcache.BigCache) {
	for message := range pc.Messages() {
		if err := handleMessage(message, cache); err != nil {
			log.Printf("处理消息失败：%v", err)
		}
	}
}

// handleMessage 处理单个消息
func handleMessage(message *sarama.ConsumerMessage, cache *bigcache.BigCache) error {
	var nginxLog NginxLog
	if err := json.Unmarshal(message.Value, &nginxLog); err != nil {
		log.Printf("二进制数据解析失败：%v", string(message.Value))
		return err
	}

	strs := strings.Split(nginxLog.Message, `"`)

	if len(strs) < 16 {
		log.Println("日志格式不正确，跳过")
		return nil
	}

	requestStatusCode := strings.Split(strs[8], " ")[2]
	statusCode, err := strconv.Atoi(requestStatusCode)
	if err != nil || statusCode < 500 {
		return nil
	}

	requestClientIP, requestUrl, requestTime, requestDuration, requestUpstreamNode := extractRequestDetails(strs)

	data := fmt.Sprintf("【网关日志5xx告警】\n请求时间：%s\n请求状态：%s\n请求耗时：%s\n后端节点：%s\n请求地址：%s\n请求链接：%s",
		requestTime, requestStatusCode, requestDuration, requestUpstreamNode, requestClientIP, requestUrl)

	key := requestClientIP + "-" + requestUrl
	cacheKey := public.GeneratorMd5(key)

	if _, err := cache.Get(cacheKey); err == nil {
		log.Println("缓存已存在，不告警：", key)
		return nil
	}

	log.Println("原始日志：", nginxLog.Message)
	fmt.Println("格式化后数据：", data)

	if err := public.Sender(data, ""); err != nil { // 移除了 people 参数
		log.Println("群机器人消息推送失败：", err)
		return err
	}

	if err := cache.Set(cacheKey, []byte(cacheKey)); err != nil {
		log.Println("写入数据到内存缓存失败：", err)
		return err
	}

	return nil
}

// extractRequestDetails 从日志字符串中提取请求的详细信息
func extractRequestDetails(strs []string) (clientIP, url, time, duration, upstreamNode string) {
	clientIP = strings.Split(strs[0], " ")[0]
	url = strs[1] + "://" + strs[3] + strs[5]
	time = strings.ReplaceAll(strings.Split(strs[0], " ")[3], "[", "")
	time = strings.Replace(time, ":", " ", 1)
	duration = strings.Split(strs[8], " ")[1]
	upstreamNode = strs[15]
	return
}
