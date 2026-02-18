#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PKGBUILD_SRC="${ROOT_DIR}/dist/aur/PKGBUILD"

AUR_DIR="${1:-}"
if [[ -z "${AUR_DIR}" ]]; then
  echo "Usage: $(basename "$0") /path/to/aur-repo" >&2
  exit 1
fi

if [[ ! -f "${PKGBUILD_SRC}" ]]; then
  echo "Missing ${PKGBUILD_SRC}. Run ./scripts/build_aur.sh first." >&2
  exit 1
fi

if [[ ! -d "${AUR_DIR}/.git" ]]; then
  echo "No git repo found at ${AUR_DIR}." >&2
  exit 1
fi

if ! command -v makepkg >/dev/null 2>&1; then
  echo "makepkg is required to generate .SRCINFO." >&2
  exit 1
fi

cp "${PKGBUILD_SRC}" "${AUR_DIR}/PKGBUILD"

pkgver="$(awk -F= '/^pkgver=/{print $2; exit}' "${AUR_DIR}/PKGBUILD")"
if [[ -z "${pkgver}" ]]; then
  echo "Could not determine pkgver from ${AUR_DIR}/PKGBUILD." >&2
  exit 1
fi

git -C "${AUR_DIR}" add PKGBUILD
(cd "${AUR_DIR}" && makepkg --printsrcinfo > .SRCINFO)
git -C "${AUR_DIR}" add .SRCINFO

if git -C "${AUR_DIR}" diff --cached --quiet; then
  echo "No AUR changes to commit."
  exit 0
fi

git -C "${AUR_DIR}" commit -m "Update to v${pkgver}"
git -C "${AUR_DIR}" push
