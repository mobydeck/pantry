#!/bin/bash
# Проверка coverage - 0 токенов
echo "🧪 Coverage..."
echo ""
if find . -name "*.go" -not -path "*/vendor/*" 2>/dev/null | head -1 | grep -q .; then
    echo "=== Go ==="
    go test ./... -cover 2>/dev/null || echo "  ❌ Ошибка"
fi
if [ -f "package.json" ] && grep -q '"jest"' package.json 2>/dev/null; then
    echo "=== Jest ==="
    npx jest --coverage --passWithNoTests 2>/dev/null | tail -15
fi
echo ""
echo "✅ Готово"
