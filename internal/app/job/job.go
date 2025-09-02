// Package job представляет процессы работающие в фоне,
// к примеру такие как сбор статистики, мониторинг или обработка задач
package job

import "github.com/sourcegraph/conc/pool"

var execPool = pool.New()
