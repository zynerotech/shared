# Общие пакеты
Набор вспомогательных общих пакетов для микросервисов.
Пакет должен быть вложен на первом уровне проекта и иметь свой go.mod и go.sum

## Использование пакетов
````go
package main

import (
	// ...
    platformconfig "gitlab.com/zynero/shared/config"
    platformdatabase "gitlab.com/zynero/shared/database"
    platformhealthcheck "gitlab.com/zynero/shared/healthcheck"
    platformlogger "gitlab.com/zynero/shared/logger"
    platformmetrics "gitlab.com/zynero/shared/metrics"
    platformserver "gitlab.com/zynero/shared/server"
	// ...
)
````

Для пакетов желательно использовать синонимы (алиасы), так как их названия могут часто повторяться

## Соблюдение обратной совместимости
Если возникнут проблемы с совместимостью, то каждый пакет можно будет подключать отдельно, но для этого потребуется использовать теги в git с префиксом в виде названия пакета.
Например, тег может быть config/v1.2.3

## Тегирование
````bash
git tag cache/0.1.4
git tag config/0.1.4
git tag database/0.1.4
git tag healthcheck/0.1.4
git tag logger/0.1.4
git tag metrics/0.1.4
git tag server/0.1.4
git tag transport/0.1.4
````