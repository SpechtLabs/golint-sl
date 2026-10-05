#!/usr/bin/env bash
# The golint-sl pre-commit hooks (see .pre-commit-hooks.yaml).
#
# Builds ./custom-gcl, golangci-lint with golint-sl baked in, from the
# .custom-gcl.yml at the repository root, then runs it in every Go module of
# the repository: the root module, and any module in a subdirectory. A
# repository with several modules and no go.mod at the root is linted module
# by module, the way `go` itself has to see it. golangci-lint finds each
# module's configuration the usual way, from the module's directory upwards.
#
#   golint-sl.sh                     lint every package of every module
#   golint-sl.sh --packages FILE...  lint only the packages FILE... are in
#
# golangci-lint comes from PATH, or through mise when it isn't on PATH.
set -euo pipefail

mode=all
if [ "${1:-}" = "--packages" ]; then
  mode=packages
  shift
fi

root="$(git rev-parse --show-toplevel)"
cd "${root}"

if [ ! -f .custom-gcl.yml ]; then
  echo "golint-sl: no .custom-gcl.yml at the repository root; see https://golint.specht-labs.de/getting-started/installation" >&2
  exit 1
fi

if command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint custom
elif command -v mise >/dev/null 2>&1; then
  mise exec -- golangci-lint custom
else
  echo "golint-sl: golangci-lint is neither on PATH nor available through mise" >&2
  exit 1
fi

# Every directory with a go.mod, as `go` would see it: modules under testdata
# and vendor, and under directories whose names start with . or _, aren't
# part of the repository's code.
modules="$(git ls-files --cached --others --exclude-standard -- go.mod '*/go.mod' |
  grep -Ev '(^|/)(testdata|vendor|[._][^/]*)/' |
  sed -e 's#/\{0,1\}go\.mod$##' -e 's#^$#.#' |
  sort -u)"

if [ -z "${modules}" ]; then
  echo "golint-sl: no go.mod in the repository; nothing to lint" >&2
  exit 0
fi

# module_of DIR prints the module DIR belongs to: the nearest directory at or
# above it that has a go.mod. It prints nothing for a directory outside every
# module.
module_of() {
  local dir="$1"
  while :; do
    if printf '%s\n' "${modules}" | grep -qxF -- "${dir}"; then
      printf '%s\n' "${dir}"
      return
    fi
    [ "${dir}" = "." ] && return
    case "${dir}" in
      */*) dir="${dir%/*}" ;;
      *) dir="." ;;
    esac
  done
}

status=0

# lint MODULE PATTERN... runs custom-gcl in MODULE, and remembers a failure
# rather than stopping, so one run reports every module's findings.
lint() {
  local module="$1"
  shift
  if ! (cd "${module}" && "${root}/custom-gcl" run "$@"); then
    status=1
  fi
}

if [ "${mode}" = all ]; then
  while IFS= read -r module; do
    lint "${module}" ./...
  done <<<"${modules}"
  exit "${status}"
fi

# Each file's package directory, relative to its module, as "module<TAB>./pkg".
pairs=""
for file in "$@"; do
  dir="$(dirname -- "${file}")"
  module="$(module_of "${dir}")"
  [ -n "${module}" ] || continue
  if [ "${dir}" = "${module}" ]; then
    pkg="."
  elif [ "${module}" = "." ]; then
    pkg="./${dir}"
  else
    pkg="./${dir#"${module}"/}"
  fi
  pairs="${pairs}${module}"$'\t'"${pkg}"$'\n'
done
pairs="$(printf '%s' "${pairs}" | sort -u)"

while IFS= read -r module; do
  [ -n "${module}" ] || continue
  pkgs=()
  while IFS=$'\t' read -r m pkg; do
    [ "${m}" = "${module}" ] && pkgs+=("${pkg}")
  done <<<"${pairs}"
  lint "${module}" "${pkgs[@]}"
done <<<"$(printf '%s\n' "${pairs}" | cut -f1 | sort -u)"

exit "${status}"
