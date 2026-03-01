#!/bin/bash
# Тестирование установки framework

echo "🧪 Тестирование Cursor Optimization Framework"
echo ""

PASS=0; FAIL=0

check() {
    local desc=$1 condition=$2
    if eval "$condition" > /dev/null 2>&1; then
        echo "  ✅ $desc"; ((PASS++))
    else
        echo "  ❌ $desc"; ((FAIL++))
    fi
}

echo "=== Структура файлов ==="
check ".cursorignore"                        "[ -f .cursorignore ]"
check ".cursor/rules/optimization.mdc"       "[ -f .cursor/rules/optimization.mdc ]"
check ".cursor/context/base.md"              "[ -f .cursor/context/base.md ]"
check ".cursor/snapshots/changes.md"         "[ -f .cursor/snapshots/changes.md ]"
check ".cursor/plans/optimization-plan.md"   "[ -f .cursor/plans/optimization-plan.md ]"
check ".cursor/plans/tasks/"                 "[ -d .cursor/plans/tasks ]"
check ".cursor/plans/done/"                  "[ -d .cursor/plans/done ]"
check ".cursor/scripts/bash/"               "[ -d .cursor/scripts/bash ]"
check ".cursor/scripts/prompts/"            "[ -d .cursor/scripts/prompts ]"

echo ""
echo "=== Промпты ==="
for name in \
    01-analyze-project \
    02-create-plan \
    03-fix-simple-bug \
    04-create-architecture \
    05-refactor \
    05b-refactor-complex \
    06-write-unit-tests \
    07-add-godoc \
    09-update-readme \
    run-next-task; do
    check "$name.txt" "[ -f .cursor/scripts/prompts/${name}.txt ]"
done

echo ""
echo "=== Bash скрипты ==="
check "analyze-project-structure.sh" "[ -f .cursor/scripts/bash/analyze-project-structure.sh ]"
check "find-todos.sh"                "[ -f .cursor/scripts/bash/find-todos.sh ]"
check "snapshot-state.sh"           "[ -f .cursor/scripts/bash/snapshot-state.sh ]"
check "check-coverage.sh"           "[ -f .cursor/scripts/bash/check-coverage.sh ]"
check "Скрипты исполняемы"          "[ -x .cursor/scripts/bash/analyze-project-structure.sh ]"

echo ""
echo "=== .cursorignore ==="
check "node_modules/"   "grep -q 'node_modules' .cursorignore"
check "vendor/"         "grep -q 'vendor/' .cursorignore"
check ".git/"           "grep -q '.git/' .cursorignore"
check ".cursor/plans/"  "grep -q '.cursor/plans/' .cursorignore"

echo ""
echo "=== Функциональный тест ==="
check "analyze-project-structure.sh запускается" "bash .cursor/scripts/bash/analyze-project-structure.sh"
check "find-todos.sh запускается"               "bash .cursor/scripts/bash/find-todos.sh"

echo ""
echo "==============================="
echo "Результат: ✅ $PASS | ❌ $FAIL"
echo ""
if [ "$FAIL" -gt 0 ]; then
    echo "⚠️  Запусти setup.sh заново или проверь ошибки выше."
    exit 1
else
    echo "🎉 Framework готов!"
    echo ""
    echo "Следующий шаг:"
    echo "  1. Отредактируй .cursor/context/base.md"
    echo "  2. В Cursor: промпт 01-analyze-project.txt (Opus)"
fi
