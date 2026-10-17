#!/usr/bin/env bash
set -euo pipefail

case "${1:?provide the target OS}" in
  linux|windows) exit 0 ;;
  darwin) ;;
  *) echo "Unsupported release OS" >&2; exit 1 ;;
esac
artifact="${2:?provide the macOS artifact path}"
if [[ "${3:?provide the GoReleaser snapshot flag}" == true ]]; then
  exit 0
fi
: "${APPLE_SIGNING_IDENTITY:?}" "${APPLE_SIGNING_KEYCHAIN:?}" "${APPLE_TEAM_ID:?}"
: "${APPLE_API_KEY_PATH:?}" "${APPLE_API_KEY_ID:?}" "${APPLE_API_ISSUER:?}"
case "$APPLE_SIGNING_IDENTITY" in
  "Developer ID Application: "*) ;;
  *) echo 'A Developer ID Application identity is required.' >&2; exit 1 ;;
esac

codesign --force --timestamp --options runtime --sign "$APPLE_SIGNING_IDENTITY" \
  --keychain "$APPLE_SIGNING_KEYCHAIN" "$artifact"
codesign --verify --deep --strict --verbose=2 "$artifact"
details="$(codesign --display --verbose=4 "$artifact" 2>&1)"
grep -Fq 'Authority=Developer ID Application: ' <<< "$details"
grep -Fxq "TeamIdentifier=$APPLE_TEAM_ID" <<< "$details"
grep -q '^Timestamp=' <<< "$details"
grep -q 'flags=.*(runtime)' <<< "$details"

work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT
submission="$work_dir/release.zip"
ditto -c -k --keepParent "$artifact" "$submission"
xcrun notarytool submit "$submission" \
  --key "$APPLE_API_KEY_PATH" --key-id "$APPLE_API_KEY_ID" --issuer "$APPLE_API_ISSUER" \
  --wait --timeout 30m --output-format json > "$work_dir/result.json"
cat "$work_dir/result.json"
jq -e '.status == "Accepted"' "$work_dir/result.json" > /dev/null
# Standalone executables cannot carry a stapled ticket.
codesign --verify --strict --check-notarization -R '=notarized' --verbose=2 "$artifact"
