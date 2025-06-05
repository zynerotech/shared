# Общие пакеты
Набор вспомогательных общих пакетов для микросервисов.
Пакет должен быть вложен на первом уровне проекта и иметь свой go.mod и go.sum

## Использование пакетов
````go
package main

import (
	// ...
    platformconfig "github.com/zynerotech/shared/config"
    platformdatabase "github.com/zynerotech/shared/database"
    platformhealthcheck "github.com/zynerotech/shared/healthcheck"
    platformlogger "github.com/zynerotech/shared/logger"
    platformmetrics "github.com/zynerotech/shared/metrics"
    platformserver "github.com/zynerotech/shared/server"
	// ...
)
````

Для пакетов желательно использовать синонимы (алиасы), так как их названия могут часто повторяться

## Соблюдение обратной совместимости
Если возникнут проблемы с совместимостью, то каждый пакет можно будет подключать отдельно, но для этого потребуется использовать теги в git с префиксом в виде названия пакета.
Например, тег может быть config/v1.2.3

## Тегирование
````bash
git tag cache/v0.1.3
git tag config/v0.1.3
git tag database/v0.1.3
git tag healthcheck/v0.1.3
git tag logger/v0.1.3
git tag metrics/v0.1.3
git tag server/v0.1.3
git tag transport/v0.1.3
````