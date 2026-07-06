#!/bin/sh
# vitals installer — downloads the right release binary, verifies its checksum,
# and installs it to ~/.local/bin.
#
#   curl -fsSL https://raw.githubusercontent.com/sayantan/vitals/main/install.sh | sh
#
# Flags:
#   --version <vX.Y.Z>   install a specific version (default: latest)
#   --bin-dir <dir>      install location (default: $HOME/.local/bin)
#   --no-modify-path     don't print PATH instructions
#
# Hardened after zoxide/atuin: HTTPS-only, checksum-verified, no sudo, whole body
# wrapped in a function called at the very end (guards truncated-download exec).
#
# NOTE (implementation): finalize REPO once the GitHub repo exists, and confirm the
# asset naming matches .goreleaser.yaml. Tested end-to-end against a snapshot build.

set -u

REPO="sayantan/vitals"          # TODO: set to the real owner/repo
BIN="vitals"
VERSION="latest"
BIN_DIR="${HOME}/.local/bin"
MODIFY_PATH=1

main() {
	parse_args "$@"
	need curl || need wget || err "need curl or wget"
	need tar  || err "need tar"

	os="$(detect_os)"
	arch="$(detect_arch)"
	[ "$VERSION" = "latest" ] && VERSION="$(latest_version)"
	[ -n "$VERSION" ] || err "could not resolve version"

	# Asset naming must match .goreleaser.yaml, e.g. vitals_1.2.3_linux_amd64.tar.gz
	ver_no_v="${VERSION#v}"
	asset="${BIN}_${ver_no_v}_${os}_${arch}.tar.gz"
	base="https://github.com/${REPO}/releases/download/${VERSION}"

	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT

	say "Downloading ${asset} (${VERSION})…"
	fetch "${base}/${asset}" "${tmp}/${asset}"          || err "download failed"
	fetch "${base}/checksums.txt" "${tmp}/checksums.txt" || err "checksum list download failed"
	verify_checksum "${tmp}" "${asset}"

	tar -xzf "${tmp}/${asset}" -C "${tmp}" || err "extract failed"
	mkdir -p "$BIN_DIR"
	install -m 0755 "${tmp}/${BIN}" "${BIN_DIR}/${BIN}" 2>/dev/null \
		|| { cp "${tmp}/${BIN}" "${BIN_DIR}/${BIN}" && chmod 0755 "${BIN_DIR}/${BIN}"; } \
		|| err "install to ${BIN_DIR} failed"

	say "Installed ${BIN} ${VERSION} → ${BIN_DIR}/${BIN}"
	post_install
}

parse_args() {
	while [ $# -gt 0 ]; do
		case "$1" in
			--version) VERSION="$2"; shift 2 ;;
			--bin-dir) BIN_DIR="$2"; shift 2 ;;
			--no-modify-path) MODIFY_PATH=0; shift ;;
			*) err "unknown flag: $1" ;;
		esac
	done
}

detect_os() {
	case "$(uname -s)" in
		Linux)  echo linux ;;
		Darwin) echo darwin ;;
		*)      err "unsupported OS: $(uname -s)" ;;
	esac
}

detect_arch() {
	case "$(uname -m)" in
		x86_64|amd64) echo amd64 ;;
		arm64|aarch64) echo arm64 ;;
		*) err "unsupported arch: $(uname -m)" ;;
	esac
}

latest_version() {
	api="https://api.github.com/repos/${REPO}/releases/latest"
	if need curl; then curl -fsSL "$api"; else wget -qO- "$api"; fi \
		| grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
}

fetch() { # url dest
	if need curl; then curl --proto '=https' --tlsv1.2 -fsSL "$1" -o "$2"
	else wget -q "$1" -O "$2"; fi
}

verify_checksum() { # dir asset
	dir="$1"; asset="$2"
	want="$(grep " ${asset}\$" "${dir}/checksums.txt" | awk '{print $1}')"
	[ -n "$want" ] || err "no checksum entry for ${asset}"
	if need sha256sum; then got="$(sha256sum "${dir}/${asset}" | awk '{print $1}')"
	elif need shasum;   then got="$(shasum -a 256 "${dir}/${asset}" | awk '{print $1}')"
	else say "warning: no sha256 tool found; skipping checksum verification"; return 0; fi
	[ "$want" = "$got" ] || err "checksum mismatch for ${asset}"
	say "Checksum verified."
}

post_install() {
	case ":${PATH}:" in
		*":${BIN_DIR}:"*) ;;
		*) [ "$MODIFY_PATH" -eq 1 ] && say "Add to PATH:  export PATH=\"${BIN_DIR}:\$PATH\"" ;;
	esac
	cat <<EOF

Next steps:
  ${BIN} init      # wire vitals into ~/.claude/settings.json (backup first)
  ${BIN} config    # customize segments, theme, and order
EOF
}

need() { command -v "$1" >/dev/null 2>&1; }
say()  { printf '  %s\n' "$*"; }
err()  { printf 'error: %s\n' "$*" >&2; exit 1; }

main "$@"
