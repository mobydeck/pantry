# Cursor Optimization Framework — Инструкция

## Установка (5 минут)

```bash
git clone https://github.com/motokazmin/cursor-optimization-framework
cd your-project-folder

# Запуск скрипта из директории фреймворка
bash ../cursor-optimization-framework/setup.sh
```

Что произойдет:

В корень вашего проекта скопируется директория .cursor/ со всеми промптами и правилами.
В корень скопируется файл test-framework.sh для диагностики.
Будет создан (или дополнен) файл .cursorignore.


Запустите диагностический скрипт, чтобы убедиться, что все компоненты скопированы
корректно и структура папок соблюдена:

```bash
bash test-framework.sh
```

Должно быть `✅ 31 | ❌ 0`.


---

## Шаг 1: Заполни base.md

Открой `.cursor/context/base.md` — там уже готовый пример для Go проекта.
Замени название и цель на свои. Структуру папок оставь если используешь
Clean Architecture, иначе обнови.

Cursor читает этот файл автоматически при каждом запросе.

---

## Шаг 2: Анализ проекта (один раз)

Открой **Cursor Composer** (`Cmd+I` / `Ctrl+I`), включи **Agent Mode**.

Скопируй `.cursor/scripts/prompts/01-analyze-project.txt` в Composer.
Модель: **claude-opus-4 + max thinking**.

Cursor сам запустит скрипты и создаст `.cursor/analysis/project-map.md`.

Стоимость: ~$1.50-3.00. Единственный раз когда используется Opus.

---

## Шаг 3: Создай план (один раз)

Скопируй `.cursor/scripts/prompts/02-create-plan.txt` в Composer.
Модель: **Sonnet БЕЗ thinking** | Agent Mode.

Cursor создаст:
- `.cursor/plans/optimization-plan.md` — индекс задач
- `.cursor/plans/tasks/task-001.md`, `task-002.md` ... — по одному файлу на задачу

---

## Шаг 4: Выполнение задач

Дальше всё сводится к одному действию — вставить один промпт и нажать Enter.

Скопируй `.cursor/scripts/prompts/run-next-task.txt` в Composer.
Модель: **та что указана в файле задачи** (обычно Haiku).
Режим: **Agent Mode**.

Cursor сам:
1. Найдёт первую незавершённую задачу в плане
2. Прочитает файл задачи (`task-NNN.md`)
3. Выполнит задачу нужным способом (баг / тесты / архитектура / документация)
4. Запустит `snapshot-state.sh` и сделает git commit
5. Переместит задачу в `done/`, отметит ✅ в индексе
6. Проверит актуальность следующей задачи

**Повторяй этот шаг до конца плана.**

---

## Весь процесс

```
setup.sh → test-framework.sh → заполни base.md
↓
01-analyze-project.txt (Opus)     → project-map.md
↓
02-create-plan.txt (Sonnet)        → tasks/task-001.md ...
↓
run-next-task.txt (Haiku)  ← повторять до конца плана
```

---

## Когда менять модель вручную

`run-next-task.txt` подбирает модель из поля "Модель" в файле задачи.
Но Cursor не переключает модель сам — это нужно сделать руками перед запуском:

| Тип задачи | Модель |
|------------|--------|
| Баги, документация | Haiku БЕЗ thinking |
| Тесты | Haiku С thinking |
| Архитектура | Sonnet БЕЗ thinking |


---

## Диагностика

**AI создаёт BUGFIX.md, SUMMARY.md**
→ Проверь `.cursor/rules/optimization.mdc`

**Задачи стоят дорого**
→ Проверь `.cursorignore`, запусти `bash test-framework.sh`

**Проверить стоимость**
→ Cursor Settings → Usage. Ориентиры: баг ~$0.03, тесты ~$0.18
