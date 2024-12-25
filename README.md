# NginxLogAlerter
一个 Kafka 消费者应用程序，它订阅特定的 Kafka 主题，解析 Nginx 日志消息，并在检测到 5xx 错误时发送告警。使用内存缓存来避免对同一请求的重复告警。
