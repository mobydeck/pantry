#!/bin/bash
# Поиск TODO/FIXME/HACK - 0 токенов
echo "🔍 Tech debt..."
echo ""
grep -rn "TODO\|FIXME\|HACK\|XXX\|BUG\|DEPRECATED" . \
    --include="*.go" --include="*.ts" --include="*.js" --include="*.py" \
    --exclude-dir=vendor --exclude-dir=node_modules --exclude-dir=.git \
    2>/dev/null | sort
echo ""
echo "✅ Готово"
