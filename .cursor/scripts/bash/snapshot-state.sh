#!/bin/bash
# Фиксация состояния после изменений

SNAPSHOT_FILE=".cursor/snapshots/changes.md"
DATE=$(date '+%Y-%m-%d %H:%M')

if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "⚠️  Git не найден. Запусти: git init"
    exit 1
fi

CHANGED=$(git diff --name-only HEAD 2>/dev/null)
STAGED=$(git diff --name-only --cached 2>/dev/null)
ALL=$(echo -e "$CHANGED\n$STAGED" | grep -v '^$' | sort -u)

echo "" >> "$SNAPSHOT_FILE"
echo "## $DATE" >> "$SNAPSHOT_FILE"

if [ -n "$ALL" ]; then
    echo "$ALL" | while read f; do
        [ -f "$f" ] && echo "- \`$f\` ($(wc -l < "$f") строк)" >> "$SNAPSHOT_FILE"
    done
    echo "---" >> "$SNAPSHOT_FILE"
    echo "📸 Зафиксировано:"
    echo "$ALL"
else
    echo "(нет изменений)" >> "$SNAPSHOT_FILE"
    echo "---" >> "$SNAPSHOT_FILE"
    echo "Нет изменений"
fi
