#!/bin/sh
# PostToolUse hook — auto-format edited api Go files with gofmt.
#
# The "go-code-style" path-rule already TELLS the model to keep gofmt-clean output;
# this hook GUARANTEES it deterministically (the harness runs it, so it can't be
# "forgotten"). It is intentionally narrow: it only touches *.go under api,
# and no-ops silently for everything else. Remove the "hooks" block from
# .claude/settings.json to disable.
#
# Claude Code passes the tool-call payload as JSON on stdin; we read tool_input.file_path.
file=$(python3 -c 'import sys,json; print(json.load(sys.stdin).get("tool_input",{}).get("file_path",""))' 2>/dev/null)

case "$file" in
  *api*/*.go)
    if [ -f "$file" ] && command -v gofmt >/dev/null 2>&1; then
      gofmt -w "$file"
    fi
    ;;
esac

exit 0
