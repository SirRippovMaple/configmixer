#!/bin/sh
# Codesigns a darwin binary with the self-signed certificate in
# MACOS_SIGN_P12 (base64-encoded p12) and MACOS_SIGN_PASSWORD.
# Called by GoReleaser as a post-build hook: macos-sign.sh <goos> <path>.
# Does nothing for non-darwin targets or when the certificate isn't set.
set -eu

goos=$1
path=$2

[ "$goos" = darwin ] || exit 0
[ -n "${MACOS_SIGN_P12:-}" ] || exit 0

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
printf %s "$MACOS_SIGN_P12" | base64 -d > "$tmp/cert.p12"
printf %s "${MACOS_SIGN_PASSWORD:-}" > "$tmp/password"

rcodesign sign \
  --p12-file "$tmp/cert.p12" \
  --p12-password-file "$tmp/password" \
  --binary-identifier "$(basename "$path")" \
  --code-signature-flags runtime \
  "$path"
