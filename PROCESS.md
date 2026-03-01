# Cursor Optimization Framework — Process

## Установка (один раз)

```bash
git clone https://github.com/motokazmin/cursor-optimization-framework
cd your-project-folder
bash ../cursor-optimization-framework/setup.sh
```

Отредактируй `.cursor/context/base.md` — замени пример на свой проект.

---

## Рабочий цикл

### Фаза 0: Анализ (один раз)

**Промпт 01** | Opus + max thinking | Agent Mode
→ Cursor сам запустит скрипты и создаст `.cursor/analysis/project-map.md`

**Промпт 02** | Sonnet БЕЗ thinking | Agent Mode
→ Создаёт индекс `.cursor/plans/optimization-plan.md`
  и отдельный файл на каждую задачу в `.cursor/plans/tasks/`

---

### Фаза 1–N: Выполнение задач

Один промпт — `run-next-task.txt` — делает всё:

```
run-next-task.txt (повторять до конца плана)
```

Cursor автоматически:
1. Найдёт первую незавершённую задачу через `grep`
2. Прочитает файл задачи через `cat`
3. Прочитает нужный промпт через `cat`
4. Прочитает целевой файл кода через `cat`
5. Выполнит задачу по инструкциям промпта
6. Запустит `snapshot-state.sh` и сделает git commit
7. Переместит задачу в `done/`, отметит ✅ в индексе
8. Проверит актуальность следующей задачи по `changes.md`

---

## Таблица промптов (используются внутри run-next-task)

| Промпт | Когда | Модель |
|--------|-------|--------|
| `03-fix-simple-bug.txt` | баг, race condition, транзакции | Haiku БЕЗ thinking |
| `04-create-architecture.txt` | слои handler/service/repository с нуля | Sonnet БЕЗ thinking |
| `05-refactor.txt` | валидация, разбить функцию, дублирование | Haiku С thinking |
| `05b-refactor-complex.txt` | интерфейсы, DI, God-object | Sonnet С thinking |
| `06-write-unit-tests.txt` | unit тесты | Haiku С thinking |
| `07-add-godoc.txt` | Godoc комментарии | Haiku С thinking |
| `09-update-readme.txt` | README | Haiku С thinking |

---

## Признаки что всё работает

- При выполнении задачи AI видит только 1 task файл, не весь план
- После выполнения файл задачи переехал в `done/`
- В `optimization-plan.md` задача отмечена ✅
- Cursor Settings → Usage: баг ~$0.03, тесты ~$0.18
