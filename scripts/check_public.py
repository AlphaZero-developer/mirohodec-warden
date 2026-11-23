#!/usr/bin/env python3
"""Public repository checks: no accidental runtime data, credentials or long Go files."""
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
FORBIDDEN_PARTS = {'.git', '.deploy', '__pycache__', 'node_modules', 'photos', 'logs', 'data'}
FORBIDDEN_NAMES = {'.env', 'config.yaml'}
SECRET_PATTERNS = {
    'telegram_token': re.compile(r'\b\d{8,11}:[A-Za-z0-9_-]{30,}\b'),
    'openrouter_key': re.compile(r'\bsk-or-v1-[A-Za-z0-9]{20,}\b'),
    'github_token': re.compile(r'\b(?:ghp_|github_pat_)[A-Za-z0-9_]{20,}\b'),
    'private_key': re.compile(r'-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----'),
}
SOURCE_EXTENSIONS = {'.go', '.py', '.yaml', '.yml', '.json', '.sql', '.md', '.html', '.js', '.css', '.sh', '.txt'}

def main() -> int:
    issues: list[str] = []
    counts = {'go': 0, 'files': 0}
    for file in ROOT.rglob('*'):
        if not file.is_file():
            continue
        relative = file.relative_to(ROOT)
        name = relative.as_posix()
        if file.name in FORBIDDEN_NAMES or any(part in FORBIDDEN_PARTS for part in relative.parts):
            issues.append(f'{name}: forbidden operational artifact')
            continue
        counts['files'] += 1
        if file.suffix not in SOURCE_EXTENSIONS:
            continue
        try:
            content = file.read_text(encoding='utf-8')
        except (UnicodeDecodeError, OSError) as exc:
            issues.append(f'{name}: cannot read text: {exc}')
            continue
        if file.suffix == '.go':
            counts['go'] += 1
            lines = len(content.splitlines())
            if lines > 200:
                issues.append(f'{name}: {lines} lines; maximum 200')
        for title, regex in SECRET_PATTERNS.items():
            if regex.search(content):
                issues.append(f'{name}: possible {title} (value withheld)')
    if issues:
        print('Public repository hygiene FAILED:')
        for issue in issues:
            print('  -', issue)
        return 1
    print(f"Public hygiene OK: {counts['go']} Go files, {counts['files']} files, <=200 Go lines/file")
    print('This scan is heuristic; still review staged changes and run a dedicated secret scanner.')
    return 0

if __name__ == '__main__':
    sys.exit(main())
