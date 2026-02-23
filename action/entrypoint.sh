#!/usr/bin/env bash
set -euo pipefail

# Omnidex GitHub Action entrypoint.
# Collects markdown files from the repository and publishes them to an Omnidex instance.

if [ -z "${OMNIDEX_URL:-}" ]; then
	echo "::error::omnidex_url input is required"
	exit 1
fi

if [ -z "${OMNIDEX_API_KEY:-}" ]; then
	echo "::error::api_key input is required"
	exit 1
fi

DOCS_PATH="${DOCS_PATH:-.}"
FILE_PATTERN="${FILE_PATTERN:-**/*.md}"
REPO="${GITHUB_REPOSITORY}"
COMMIT_SHA="${GITHUB_SHA}"

API_URL="${OMNIDEX_URL%/}/api/v1/docs"

echo "Publishing docs from '${DOCS_PATH}' to ${OMNIDEX_URL}"
echo "Repository: ${REPO}"
echo "Commit: ${COMMIT_SHA}"

# Find all matching documentation files.
cd "${GITHUB_WORKSPACE}/${DOCS_PATH}"

# Build the JSON payload with all documents.
DOCUMENTS="[]"
FILE_COUNT=0

while IFS= read -r -d '' file; do
	# Get the relative path from the docs directory.
	rel_path="${file#./}"

	# Read file content.
	content=$(cat "$file")

	# Add document to the array using jq.
	DOCUMENTS=$(echo "$DOCUMENTS" | jq \
		--arg path "$rel_path" \
		--arg content "$content" \
		'. += [{"path": $path, "content": $content, "action": "upsert"}]')

	FILE_COUNT=$((FILE_COUNT + 1))
done < <(find . -name "*.md" -type f -print0 | sort -z)

if [ "$FILE_COUNT" -eq 0 ]; then
	echo "::warning::No markdown files found in '${DOCS_PATH}'"
	exit 0
fi

echo "Found ${FILE_COUNT} documentation file(s)"

# Build the request payload.
PAYLOAD=$(jq -n \
	--arg repo "$REPO" \
	--arg commit_sha "$COMMIT_SHA" \
	--argjson documents "$DOCUMENTS" \
	'{"repo": $repo, "commit_sha": $commit_sha, "documents": $documents}')

# Send the request.
HTTP_RESPONSE=$(curl -s -w "\n%{http_code}" \
	-X POST "$API_URL" \
	-H "Authorization: Bearer ${OMNIDEX_API_KEY}" \
	-H "Content-Type: application/json" \
	-d "$PAYLOAD")

HTTP_BODY=$(echo "$HTTP_RESPONSE" | head -n -1)
HTTP_STATUS=$(echo "$HTTP_RESPONSE" | tail -n 1)

if [ "$HTTP_STATUS" -ge 200 ] && [ "$HTTP_STATUS" -lt 300 ]; then
	echo "Successfully published documentation"
	echo "$HTTP_BODY" | jq . 2>/dev/null || echo "$HTTP_BODY"
else
	echo "::error::Failed to publish documentation (HTTP ${HTTP_STATUS})"
	echo "$HTTP_BODY"
	exit 1
fi
