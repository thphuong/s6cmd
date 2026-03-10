#!/usr/bin/env bash
# e2e_test.sh — End-to-end tests for s6cmd against any S3-compatible provider.
#
# Usage:
#   ./e2e_test.sh --profile <aws-profile> [--bucket <bucket-name>] [--skip <tests>]
#
# Options:
#   --profile <name>   AWS profile to use (required)
#   --bucket <name>    Test bucket name (default: s6cmd-e2e-test)
#   --skip <tests>     Comma-separated list of test names to skip
#   --binary <path>    Path to s6cmd binary (default: ./s6cmd)
#   --help             Show this help
#
# Skippable test names:
#   mb          Make bucket
#   rb          Remove bucket
#   cp-upload   Upload file
#   cp-download Download file
#   ls-buckets  List buckets
#   ls-objects  List objects
#   cat         Cat object contents
#   mv          Move/rename object
#   head        Head object metadata
#   du          Disk usage
#   rm          Delete object
#   presign-get Presign GET + curl verify
#   presign-put Presign PUT + curl upload + verify
#   presign-del Presign DELETE + curl delete + verify
#   version     Version command
#
# Examples:
#   ./e2e_test.sh --profile local
#   ./e2e_test.sh --profile aws-prod --skip mb,rb
#   ./e2e_test.sh --profile minio --bucket my-test-bucket --skip presign-del

set -uo pipefail

# --- Defaults ---
PROFILE=""
BUCKET="s6cmd-e2e-test"
SKIP=""
BINARY="./s6cmd"
TMPDIR_E2E=""
PASSED=0
FAILED=0
SKIPPED=0

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
NC='\033[0m'

# --- Helpers ---
usage() {
  sed -n '2,/^$/{ s/^# \?//; p }' "$0"
  exit 0
}

die() { echo -e "${RED}ERROR: $1${NC}" >&2; exit 1; }

s6() { "$BINARY" --profile "$PROFILE" "$@"; }

is_skipped() {
  [[ ",$SKIP," == *",$1,"* ]]
}

run_test() {
  local name="$1"
  local desc="$2"
  shift 2

  if is_skipped "$name"; then
    echo -e "  ${YELLOW}SKIP${NC} $desc"
    ((SKIPPED++))
    return 0
  fi

  if "$@" >/dev/null 2>&1; then
    echo -e "  ${GREEN}PASS${NC} $desc"
    ((PASSED++))
  else
    echo -e "  ${RED}FAIL${NC} $desc"
    ((FAILED++))
  fi
  return 0
}

# Like run_test but captures output for further assertions.
run_test_output() {
  local name="$1"
  local desc="$2"
  shift 2

  if is_skipped "$name"; then
    echo -e "  ${YELLOW}SKIP${NC} $desc"
    ((SKIPPED++))
    TEST_OUTPUT=""
    return 0
  fi

  TEST_OUTPUT=""
  if TEST_OUTPUT=$("$@" 2>&1); then
    echo -e "  ${GREEN}PASS${NC} $desc"
    ((PASSED++))
  else
    echo -e "  ${RED}FAIL${NC} $desc (output: $TEST_OUTPUT)"
    ((FAILED++))
  fi
  return 0
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local desc="$3"

  if [[ "$haystack" == *"$needle"* ]]; then
    echo -e "  ${GREEN}PASS${NC} $desc"
    ((PASSED++))
  else
    echo -e "  ${RED}FAIL${NC} $desc (expected '$needle' in output)"
    ((FAILED++))
  fi
}

assert_not_contains() {
  local haystack="$1"
  local needle="$2"
  local desc="$3"

  if [[ "$haystack" != *"$needle"* ]]; then
    echo -e "  ${GREEN}PASS${NC} $desc"
    ((PASSED++))
  else
    echo -e "  ${RED}FAIL${NC} $desc (unexpected '$needle' in output)"
    ((FAILED++))
  fi
}

cleanup() {
  echo ""
  echo -e "${BOLD}Cleanup${NC}"

  # Delete all objects in test bucket (best-effort)
  if ! is_skipped "rm"; then
    s6 rm "s3://$BUCKET/*" 2>/dev/null || true
    echo "  Deleted objects in s3://$BUCKET"
  fi

  # Remove bucket (best-effort)
  if ! is_skipped "rb"; then
    s6 rb "s3://$BUCKET" 2>/dev/null || true
    echo "  Removed bucket s3://$BUCKET"
  fi

  # Remove temp files
  if [[ -n "$TMPDIR_E2E" && -d "$TMPDIR_E2E" ]]; then
    rm -rf "$TMPDIR_E2E"
    echo "  Removed temp dir $TMPDIR_E2E"
  fi
}

# --- Parse args ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile) PROFILE="$2"; shift 2 ;;
    --bucket)  BUCKET="$2"; shift 2 ;;
    --skip)    SKIP="$2"; shift 2 ;;
    --binary)  BINARY="$2"; shift 2 ;;
    --help|-h) usage ;;
    *) die "Unknown option: $1" ;;
  esac
done

[[ -n "$PROFILE" ]] || die "--profile is required"
[[ -x "$BINARY" ]]  || die "Binary not found or not executable: $BINARY"

TMPDIR_E2E=$(mktemp -d)
echo "hello s6cmd" > "$TMPDIR_E2E/test1.txt"
echo "second file" > "$TMPDIR_E2E/test2.txt"

# --- Run tests ---
echo -e "${BOLD}s6cmd e2e tests${NC}"
echo -e "  Profile: $PROFILE"
echo -e "  Bucket:  $BUCKET"
echo -e "  Binary:  $BINARY"
[[ -n "$SKIP" ]] && echo -e "  Skip:    $SKIP" || true
echo ""

# Register cleanup trap after args are parsed
trap cleanup EXIT

# -- version --
echo -e "${BOLD}[version]${NC}"
run_test "version" "version command" s6 version

# -- mb --
echo -e "${BOLD}[mb] Make bucket${NC}"
run_test "mb" "create test bucket s3://$BUCKET" s6 mb "s3://$BUCKET"

# -- ls-buckets --
echo -e "${BOLD}[ls-buckets] List buckets${NC}"
run_test_output "ls-buckets" "list all buckets" s6 ls
if [[ -n "$TEST_OUTPUT" ]]; then
  assert_contains "$TEST_OUTPUT" "$BUCKET" "  bucket $BUCKET appears in listing"
fi

# -- cp-upload --
echo -e "${BOLD}[cp-upload] Upload files${NC}"
run_test "cp-upload" "upload test1.txt" s6 cp "$TMPDIR_E2E/test1.txt" "s3://$BUCKET/test1.txt"
run_test "cp-upload" "upload test2.txt to subdir" s6 cp "$TMPDIR_E2E/test2.txt" "s3://$BUCKET/subdir/test2.txt"

# -- ls-objects --
echo -e "${BOLD}[ls-objects] List objects${NC}"
run_test_output "ls-objects" "list objects in bucket" s6 ls "s3://$BUCKET/"
if [[ -n "$TEST_OUTPUT" ]]; then
  assert_contains "$TEST_OUTPUT" "test1.txt" "  test1.txt in listing"
  assert_contains "$TEST_OUTPUT" "subdir/" "  subdir/ in listing"
fi

# -- cat --
echo -e "${BOLD}[cat] Read object${NC}"
run_test_output "cat" "cat test1.txt" s6 cat "s3://$BUCKET/test1.txt"
if [[ -n "$TEST_OUTPUT" ]]; then
  assert_contains "$TEST_OUTPUT" "hello s6cmd" "  content matches"
fi

# -- cp-download --
echo -e "${BOLD}[cp-download] Download file${NC}"
run_test "cp-download" "download test1.txt" s6 cp "s3://$BUCKET/test1.txt" "$TMPDIR_E2E/downloaded.txt"
if ! is_skipped "cp-download"; then
  DOWNLOADED=$(cat "$TMPDIR_E2E/downloaded.txt" 2>/dev/null || echo "")
  assert_contains "$DOWNLOADED" "hello s6cmd" "  downloaded content matches"
fi

# -- mv --
echo -e "${BOLD}[mv] Move object${NC}"
run_test "mv" "rename test1.txt to moved.txt" s6 mv "s3://$BUCKET/test1.txt" "s3://$BUCKET/moved.txt"
if ! is_skipped "mv"; then
  run_test_output "ls-objects" "verify moved.txt exists" s6 ls "s3://$BUCKET/"
  if [[ -n "$TEST_OUTPUT" ]]; then
    assert_contains "$TEST_OUTPUT" "moved.txt" "  moved.txt in listing"
    assert_not_contains "$TEST_OUTPUT" "test1.txt" "  test1.txt no longer in listing"
  fi
fi

# -- head --
echo -e "${BOLD}[head] Object metadata${NC}"
run_test_output "head" "head moved.txt" s6 head "s3://$BUCKET/moved.txt"
if [[ -n "$TEST_OUTPUT" ]]; then
  assert_contains "$TEST_OUTPUT" "content_type" "  has content_type field"
  assert_contains "$TEST_OUTPUT" "size" "  has size field"
fi

# -- du --
echo -e "${BOLD}[du] Disk usage${NC}"
run_test_output "du" "disk usage of bucket" s6 du "s3://$BUCKET/*"
if [[ -n "$TEST_OUTPUT" ]]; then
  assert_contains "$TEST_OUTPUT" "bytes" "  reports bytes"
  assert_contains "$TEST_OUTPUT" "objects" "  reports object count"
fi

# -- presign-get --
echo -e "${BOLD}[presign-get] Presign GET${NC}"
if ! is_skipped "presign-get"; then
  PRESIGN_URL=$(s6 presign "s3://$BUCKET/subdir/test2.txt" 2>&1)
  if [[ -n "$PRESIGN_URL" ]]; then
    echo -e "  ${GREEN}PASS${NC} generate presign GET URL"
    ((PASSED++))
    CURL_OUT=$(curl -sf "$PRESIGN_URL" 2>&1 || echo "__CURL_FAIL__")
    if [[ "$CURL_OUT" == *"second file"* ]]; then
      echo -e "  ${GREEN}PASS${NC} curl GET returns correct content"
      ((PASSED++))
    else
      echo -e "  ${RED}FAIL${NC} curl GET content mismatch (got: $CURL_OUT)"
      ((FAILED++))
    fi
  else
    echo -e "  ${RED}FAIL${NC} generate presign GET URL"
    ((FAILED++))
  fi
else
  echo -e "  ${YELLOW}SKIP${NC} presign GET"
  ((SKIPPED++))
fi

# -- presign-put --
echo -e "${BOLD}[presign-put] Presign PUT${NC}"
if ! is_skipped "presign-put"; then
  PRESIGN_URL=$(s6 presign --method PUT "s3://$BUCKET/presign-uploaded.txt" 2>&1)
  if [[ -n "$PRESIGN_URL" ]]; then
    echo -e "  ${GREEN}PASS${NC} generate presign PUT URL"
    ((PASSED++))
    curl -sf -X PUT -d "uploaded via presign" "$PRESIGN_URL" >/dev/null 2>&1
    VERIFY=$(s6 cat "s3://$BUCKET/presign-uploaded.txt" 2>&1 || echo "")
    if [[ "$VERIFY" == *"uploaded via presign"* ]]; then
      echo -e "  ${GREEN}PASS${NC} PUT upload + content verify"
      ((PASSED++))
    else
      echo -e "  ${RED}FAIL${NC} PUT content mismatch (got: $VERIFY)"
      ((FAILED++))
    fi
  else
    echo -e "  ${RED}FAIL${NC} generate presign PUT URL"
    ((FAILED++))
  fi
else
  echo -e "  ${YELLOW}SKIP${NC} presign PUT"
  ((SKIPPED++))
fi

# -- presign-del --
echo -e "${BOLD}[presign-del] Presign DELETE${NC}"
if ! is_skipped "presign-del"; then
  # Ensure file exists for delete test
  s6 cp "$TMPDIR_E2E/test1.txt" "s3://$BUCKET/presign-delete-me.txt" >/dev/null 2>&1
  PRESIGN_URL=$(s6 presign --method DELETE "s3://$BUCKET/presign-delete-me.txt" 2>&1)
  if [[ -n "$PRESIGN_URL" ]]; then
    echo -e "  ${GREEN}PASS${NC} generate presign DELETE URL"
    ((PASSED++))
    curl -sf -X DELETE "$PRESIGN_URL" >/dev/null 2>&1 || true
    # Verify object is gone
    LISTING=$(s6 ls "s3://$BUCKET/" 2>&1 || echo "")
    if [[ "$LISTING" != *"presign-delete-me.txt"* ]]; then
      echo -e "  ${GREEN}PASS${NC} DELETE removed object"
      ((PASSED++))
    else
      echo -e "  ${RED}FAIL${NC} object still exists after DELETE"
      ((FAILED++))
    fi
  else
    echo -e "  ${RED}FAIL${NC} generate presign DELETE URL"
    ((FAILED++))
  fi
else
  echo -e "  ${YELLOW}SKIP${NC} presign DELETE"
  ((SKIPPED++))
fi

# -- rm --
echo -e "${BOLD}[rm] Delete objects${NC}"
run_test "rm" "delete all objects in bucket" s6 rm "s3://$BUCKET/*"

# -- rb --
echo -e "${BOLD}[rb] Remove bucket${NC}"
run_test "rb" "remove test bucket" s6 rb "s3://$BUCKET"

# --- Summary ---
# Disable cleanup trap since rm/rb tests already cleaned up
trap - EXIT
# Clean temp files only
[[ -n "$TMPDIR_E2E" && -d "$TMPDIR_E2E" ]] && rm -rf "$TMPDIR_E2E"

echo ""
echo -e "${BOLD}Results${NC}"
TOTAL=$((PASSED + FAILED + SKIPPED))
echo -e "  Total: $TOTAL | ${GREEN}Passed: $PASSED${NC} | ${RED}Failed: $FAILED${NC} | ${YELLOW}Skipped: $SKIPPED${NC}"

if [[ $FAILED -gt 0 ]]; then
  exit 1
fi
