# Gomodinspect

## Описание
Gomodinspect -- инспектор зависимостей github репозиториев. Для указанного репозитория выводит данные о модуле и список зависимостей, которые можно обновить. CLI-программа, которая на вход получает адрес github репозитория, а на выходе -- имя модуля, версия golang и список зависимостей для обновления.

## Сборка, запуск и использование
### Локально
- Можно через make (билдить не надо):
    ```
    make run REPO=https://github.com/gin-gonic/gin
    ```
- Можно сбилдить через make, а потом запускать бинарник:
    ```
    > make build
    > ./bin/gomodinspect https://github.com/gin-gonic/gin
    ```
### Через докер
```bash
make docker-run REPO=<repo-url>
```

## Конфигурация
Конфигурационный файл находится в корне репозитория `config.yaml`:
```yaml
redis:
  addr: "${REDIS_ADDR}"
  password: "${REDIS_PASSWORD}"
  db: 0

github:
  token: "${GITHUB_TOKEN}"

log:
  level: "info"
```

Можно указать фактические значения или написать название переменной окружения таким образом как в примере: `${ENV_VAR_NAME}`. Во втором случае в конфиг будут подставлены значения соответствующих переменных из окружения.\
При этом для удобства можно указать переменные в файле .env
> [!TIP]
> Реальные переменные окружения имеют приоритет над файлом `.env`

Для быстрого старта вам достаточно скопировать `.env.example` в `.env` и вставить свой github api токен\
[https://docs.github.com/en/rest/authentication/authenticating-to-the-rest-api?apiVersion=2022-11-28#authenticating-with-a-personal-access-token](Подробнее где получить этот токен)


## Архитектура
Я взял за основу гексагональную архитектуру

Структура кода:
```
├── cmd
│   └── gomodinspect
│       └── main.go
├── internal
│   ├── adapters
│   │   ├── primary             // адаптеры входящих подключений
│   │   │   └── cli             // на данный момент только консольный интерфейс
│   │   │       └── runner.go   
│   │   └── secondary           // адаптеры входящих подключений
│   │       ├── github          // github api
│   │       │   └── fetcher.go  
│   │       ├── goproxy         // GOPROXY protocol для получения актуальной latest версии
│   │       │   └── checker.go
│   │       └── redis           // кеширование через redis
│   │           └── cache.go
│   ├── config                  // код для парсинга конфига
│   │   └── config.go
│   ├── core
│   │   ├── domain              // доменные модели
│   │   │   ├── analysis.go
│   │   │   └── module.go
│   │   └── services            // бизнес-логика
│   │       └── inspector.go
│   ├── logger
│   │   └── logger.go
│   └── ports                   // интерфейсы для IoC
│       ├── analysis_cache.go   // интерфейс для кеширования
│       ├── inspector.go        // интерфейс сервиса inspector
│       ├── repo_fetcher.go     // интерфейс для сервиса получения go.mod с репозитория
│       └── version_checker.go  // интерфейс для сервиса получения latest версий
```

В моем случае `core` сервис это:
```go
type Inspector struct {
	repoFetcher    ports.RepoFetcher
	versionChecker ports.VersionChecker
	cache          ports.AnalysisCache
	logger         *slog.Logger
}
```
который реализует интерфейс с единственным методом:
```go
type Inspector interface {
	Inspect(ctx context.Context, repoURL string) (*domain.ModuleInfo, error)
}
```
В `core/domain/` лежат доменные сущности, которые ни от чего не зависят: 
```go
// AnalysisRecord представляет модель результата анализа для конкретного репозитория
type AnalysisRecord struct {
	RepoURL    string    `json:"repo_url"`
	ModuleName string    `json:"module_name"`
	GoVersion  string    `json:"go_version"`
	AnalyzedAt time.Time `json:"analyzed_at"`
	DepsJSON   string    `json:"deps_json"`
}

// ModuleInfo содержит основную информацию о Go-модуле
type ModuleInfo struct {
	Name      string       `json:"name"`
	GoVersion string       `json:"go_version"`
	Deps      []Dependency `json:"dependencies"`
}

// Dependency описывает одну зависимость Go-модуля.
type Dependency struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	UpdateAvail    bool   `json:"update_available"`
}

```






## TODO
- [ ] метод Inspect() сделать чтобы параллельно обрабатывал запросы по разным зависимостям
- [ ] дописать readme.md
- [ ] in-memory cache
- [ ] clear
- [ ] gh token
- [ ] добавить in-memory cache
- [ ] unit tests
- [ ] ci
- [ ] gh badges
