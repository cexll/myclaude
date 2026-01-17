#!/bin/bash
# SPARV Changelog Update Script
# Adds entries to .sparv/CHANGELOG.md under [Unreleased] section

set -e

usage() {
	cat <<'EOF'
Usage: changelog-update.sh --type <TYPE> --desc "description" [--file PATH]

Adds a changelog entry under [Unreleased] section.

Options:
  --type TYPE   Change type: Added|Changed|Fixed|Removed
  --desc DESC   Description of the change
  --file PATH   Custom changelog path (default: .sparv/CHANGELOG.md)

Examples:
  changelog-update.sh --type Added --desc "User authentication module"
  changelog-update.sh --type Fixed --desc "Login timeout issue"
EOF
}

CHANGELOG=".sparv/CHANGELOG.md"
TYPE=""
DESC=""

while [ $# -gt 0 ]; do
	case "$1" in
	-h|--help) usage; exit 0 ;;
	--type) TYPE="$2"; shift 2 ;;
	--desc) DESC="$2"; shift 2 ;;
	--file) CHANGELOG="$2"; shift 2 ;;
	*) usage >&2; exit 1 ;;
	esac
done

# Validate inputs
if [ -z "$TYPE" ] || [ -z "$DESC" ]; then
	echo "❌ Error: --type and --desc are required" >&2
	usage >&2
	exit 1
fi

# Validate type
case "$TYPE" in
Added|Changed|Fixed|Removed) ;;
*)
	echo "❌ Error: Invalid type '$TYPE'. Must be: Added|Changed|Fixed|Removed" >&2
	exit 1
	;;
esac

# Check changelog exists
if [ ! -f "$CHANGELOG" ]; then
	echo "❌ Error: Changelog not found: $CHANGELOG" >&2
	echo "   Run init-session.sh first to create it." >&2
	exit 1
fi

# Check if [Unreleased] section exists
if ! grep -q "## \[Unreleased\]" "$CHANGELOG"; then
	echo "❌ Error: [Unreleased] section not found in $CHANGELOG" >&2
	exit 1
fi

# Check if the type section already exists under [Unreleased]
# We need to insert after [Unreleased] but before the next ## section
TEMP_FILE=$(mktemp)
trap "rm -f $TEMP_FILE" EXIT

# Find if ### $TYPE exists between [Unreleased] and next ## section
IN_UNRELEASED=0
TYPE_FOUND=0
TYPE_LINE=0
UNRELEASED_LINE=0
NEXT_SECTION_LINE=0

line_num=0
while IFS= read -r line; do
	((line_num++))
	if [[ "$line" =~ ^##[[:space:]]\[Unreleased\] ]]; then
		IN_UNRELEASED=1
		UNRELEASED_LINE=$line_num
	elif [[ $IN_UNRELEASED -eq 1 && "$line" =~ ^##[[:space:]] && ! "$line" =~ ^###[[:space:]] ]]; then
		NEXT_SECTION_LINE=$line_num
		break
	elif [[ $IN_UNRELEASED -eq 1 && "$line" =~ ^###[[:space:]]$TYPE ]]; then
		TYPE_FOUND=1
		TYPE_LINE=$line_num
	fi
done < "$CHANGELOG"

if [ $TYPE_FOUND -eq 1 ]; then
	# Append under existing ### $TYPE section
	awk -v type_line="$TYPE_LINE" -v desc="$DESC" '
		NR == type_line { print; getline; print; print "- " desc; next }
		{ print }
	' "$CHANGELOG" > "$TEMP_FILE"
else
	# Create new ### $TYPE section after [Unreleased]
	awk -v unreleased_line="$UNRELEASED_LINE" -v type="$TYPE" -v desc="$DESC" '
		NR == unreleased_line { print; print ""; print "### " type; print "- " desc; next }
		{ print }
	' "$CHANGELOG" > "$TEMP_FILE"
fi

mv "$TEMP_FILE" "$CHANGELOG"

echo "✅ Added to $CHANGELOG:"
echo "   ### $TYPE"
echo "   - $DESC"
