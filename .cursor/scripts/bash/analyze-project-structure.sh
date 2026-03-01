#!/bin/bash
# Анализ структуры проекта - 0 токенов

echo "🔍 Анализ структуры проекта"
echo "Дата: $(date '+%Y-%m-%d %H:%M')"
echo ""

count_lines() {
    local pattern=$1 name=$2
    local count=$(find . -name "$pattern" \
        -not -path "*/node_modules/*" -not -path "*/vendor/*" \
        -not -path "*/dist/*" -not -path "*/build/*" \
        -not -path "*/.git/*" -not -path "*/target/*" \
        -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')
    [ -n "$count" ] && [ "$count" != "0" ] && echo "  $name: $count строк"
}

count_files() {
    local pattern=$1 name=$2
    local count=$(find . -name "$pattern" \
        -not -path "*/node_modules/*" -not -path "*/vendor/*" \
        -not -path "*/dist/*" -not -path "*/build/*" \
        -not -path "*/.git/*" -not -path "*/target/*" \
        2>/dev/null | wc -l)
    [ "$count" != "0" ] && echo "  $name: $count файлов"
}

echo "=== Файлы ==="
count_files "*.go" "Go"
count_files "*.ts" "TypeScript"
count_files "*.js" "JavaScript"
count_files "*.py" "Python"
count_files "*.java" "Java"
count_files "*.rs" "Rust"

echo ""
echo "=== Строки кода ==="
count_lines "*.go" "Go"
count_lines "*.ts" "TypeScript"
count_lines "*.js" "JavaScript"
count_lines "*.py" "Python"

echo ""
echo "=== Тесты ==="
test_count=$(find . \( -name "*_test.go" -o -name "*.test.ts" -o -name "*.test.js" \
    -o -name "*.spec.ts" -o -name "*.spec.js" -o -name "test_*.py" \) \
    -not -path "*/node_modules/*" -not -path "*/vendor/*" 2>/dev/null | wc -l)
if [ "$test_count" != "0" ]; then
    echo "  ✅ Тестовых файлов: $test_count"
    find . \( -name "*_test.go" -o -name "*.test.ts" \) \
        -not -path "*/node_modules/*" -not -path "*/vendor/*" 2>/dev/null
else
    echo "  ❌ Тестов не найдено"
fi

echo ""
echo "=== Крупные файлы Go (>150 строк) ==="
find . -name "*.go" -not -path "*/vendor/*" -exec wc -l {} \; 2>/dev/null | \
    awk '$1 > 150 {print "  ⚠️  "$1" строк: "$2}' | sort -rn

echo ""
echo "=== TODO / Tech debt ==="
todo_count=$(grep -r "TODO\|FIXME\|HACK\|XXX" . \
    --include="*.go" --include="*.ts" --include="*.js" \
    --exclude-dir=vendor --exclude-dir=.git 2>/dev/null | wc -l)
echo "  Найдено: $todo_count"
grep -rn "TODO\|FIXME\|HACK" . \
    --include="*.go" --exclude-dir=vendor --exclude-dir=.git 2>/dev/null | head -10

echo ""
echo "✅ Готово"
