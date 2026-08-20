#!/bin/bash
# Validate commit message format and quality

commit_msg_file=$1

if [ ! -f "$commit_msg_file" ]; then
  exit 0
fi

commit_msg=$(head -n 1 "$commit_msg_file")
commit_msg_len=${#commit_msg}

# Rule 1: Commit message must be at least 10 characters
if [ "$commit_msg_len" -lt 10 ]; then
  echo "❌ Commit message too short (${commit_msg_len} chars, minimum 10)"
  echo "   Your message: $commit_msg"
  exit 1
fi

# Rule 2: Avoid vague messages
if [[ "$commit_msg" =~ ^[Ff]ix[[:space:]]$|^[Uu]pdate[[:space:]]$|^[Cc]hange[[:space:]]$|^[Ww]ork[[:space:]] ]]; then
  echo "❌ Commit message too vague"
  echo "   Avoid: 'Fix', 'Update', 'Change', 'Work'"
  echo "   Better: 'Fix race condition in feature X' or 'Update docs for API changes'"
  exit 1
fi

# Rule 3: Start with capital letter (except for emojis)
if ! [[ "$commit_msg" =~ ^[A-Z🔧🐛✨📚🎯]|^[Cc]onfig ]]; then
  echo "❌ Commit message should start with capital letter"
  echo "   Your message: $commit_msg"
  exit 1
fi

# Rule 4: No period at end of first line (unless it's an abbreviation)
if [[ "$commit_msg" =~ \.$  ]] && ! [[ "$commit_msg" =~ [A-Z]\.[A-Z]\. ]]; then
  echo "⚠️  First line shouldn't end with period"
  echo "   Your message: $commit_msg"
fi

# Rule 5: Imperative mood (preferred for commits)
if [[ "$commit_msg" =~ ^[A-Z][a-z]+ed[[:space:]]|^[A-Z][a-z]+ing[[:space:]] ]]; then
  echo "⚠️  Consider using imperative mood (e.g., 'Add' not 'Added', 'Fix' not 'Fixing')"
  echo "   Your message: $commit_msg"
  read -p "Continue? (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
  fi
fi

# Rule 6: Reference issue number if fixing something
if [[ "$commit_msg" =~ [Ff]ix|[Rr]esolve|[Cc]lose ]]; then
  if ! [[ "$commit_msg" =~ \#[0-9] ]]; then
    echo "ℹ️  Consider referencing an issue number (e.g., 'Fix bug in #123')"
    read -p "Continue? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      exit 1
    fi
  fi
fi

exit 0
