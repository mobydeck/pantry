# Контекст проекта

## О проекте
**Название:** pantry
**Цель:** Local note storage for coding agents. Your agent keeps notes on decisions, bugs, and context across sessions — no cloud, no API keys required, no cost.

## Стек
- Backend: Go <unknown version>
- Тестирование: testify, gomock
- Конфиг: godotenv, viper

## Архитектура
Clean Architecture: handler → service → repository

```
cmd/pantry/
└── main.go
internal/
├── models/
└── config/
pkg/
└── cli/
```

## Соглашения
- Git: Conventional Commits (feat/fix/refactor/docs/test/chore)
- Ошибки: всегда fmt.Errorf("context: %w", err), не игнорировать
- Логирование: structured logging, zerolog
- Интерфейсы: определять в пакете потребителя (не в repository)
- Тесты: testify/assert, моки через gomock или вручную
- Минимум coverage: 60% для service/, 40% для handler/
- go vet + golangci-lint перед коммитом

## Исключения (не трогать)
- vendor/ — зависимости
- .env — секреты, не коммитить
