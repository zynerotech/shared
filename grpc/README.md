# gRPC пакет

Пакет предоставляет вспомогательные функции для запуска gRPC сервера и создания клиента с поддержкой middleware. Внутри реализованы interceptors для логирования через `@/logger` и экспорта метрик в Prometheus.

## Использование

### Сервер
```go
l := logger.New()
cfg := grpc.Config{
    Address:            ":50051",
    Timeout:            10 * time.Second,
    KeepAliveTime:      10 * time.Second,
    KeepAliveTimeout:   2 * time.Second,
    EnforcementMinTime: 5 * time.Second,
}

srv, _ := grpc.NewServer(cfg, l)

// регистрируем сервисы
pb.RegisterMyServiceServer(srv.GRPCServer(), myImpl)

// запускаем
if err := srv.Start(); err != nil {
    log.Fatal(err)
}
```

### Пример конфигурации YAML
```yaml
grpc:
  enabled: true
  address: ":50051"
  timeout: 10s
  tls_cert_file: ""
  tls_key_file: ""
  max_connection_age: 7200s
  max_connection_age_grace: 30s
  keep_alive_time: 10s
  keep_alive_timeout: 2s
  enforcement_min_time: 5s
  enforcement_permit: true
```

### Клиент
```go
ctx := context.Background()
client, _ := grpc.Dial(ctx, "localhost:50051")
defer client.Close()

myClient := pb.NewMyServiceClient(client.Conn())
resp, err := myClient.MyMethod(ctx, &pb.MyRequest{})
```

