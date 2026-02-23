#!/usr/bin/env bash
set -euo pipefail

# Seed script for local development.
# Publishes sample documentation to a running Omnidex instance.

OMNIDEX_URL="${OMNIDEX_URL:-http://localhost:8080}"
OMNIDEX_API_KEY="${OMNIDEX_API_KEY:-changeme}"
DOCS_PATH="${DOCS_PATH:-docs/sample}"
REPO_NAME="${REPO_NAME:-omnidex/omnidex}"
COMMIT_SHA="${COMMIT_SHA:-local}"

if [ ! -d "${DOCS_PATH}" ]; then
	echo "Error: docs directory '${DOCS_PATH}' does not exist"
	exit 1
fi

API_URL="${OMNIDEX_URL%/}/api/v1/docs"

echo "Seeding docs from '${DOCS_PATH}' to ${OMNIDEX_URL}"
echo "Repository: ${REPO_NAME}"

# Build the JSON payload with all documents.
DOCUMENTS="[]"
FILE_COUNT=0

while IFS= read -r -d '' file; do
	rel_path="${file#./}"
	content=$(cat "$file")

	DOCUMENTS=$(echo "$DOCUMENTS" | jq \
		--arg path "$rel_path" \
		--arg content "$content" \
		'. += [{"path": $path, "content": $content, "action": "upsert"}]')

	FILE_COUNT=$((FILE_COUNT + 1))
done < <(cd "${DOCS_PATH}" && find . -name "*.md" -type f -print0 | sort -z)

if [ "$FILE_COUNT" -eq 0 ]; then
	echo "Error: No markdown files found in '${DOCS_PATH}'"
	exit 1
fi

echo "Found ${FILE_COUNT} documentation file(s)"

# Build the request payload.
PAYLOAD=$(jq -n \
	--arg repo "$REPO_NAME" \
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
	echo "Successfully seeded documentation"
	echo "$HTTP_BODY" | jq . 2>/dev/null || echo "$HTTP_BODY"
else
	echo "Error: Failed to seed documentation (HTTP ${HTTP_STATUS})"
	echo "$HTTP_BODY"
	exit 1
fi
