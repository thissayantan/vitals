#!/bin/sh
# vitals installer — downloads the right release binary, verifies its checksum,
# and installs it to ~/.local/bin.
#
#   curl -fsSL https://raw.githubusercontent.com/thissayantan/vitals/main/install.sh | sh
#
# Flags:
#   --version <vX.Y.Z>   install a specific version (default: latest)
#   --bin-dir <dir>      install location (default: $HOME/.local/bin)
#   --charset <name>     seed a config with this charset if none exists
#                        (auto|unicode|nerdfont|ascii); enables icons on install
#   --nerdfont           shorthand for --charset nerdfont (language/status icons)
#   --no-modify-path     don't print PATH instructions
#
# Hardened after zoxide/atuin: HTTPS-only, checksum-verified, no sudo, whole body
# wrapped in a function called at the very end (guards truncated-download exec).

set -u

REPO="thissayantan/vitals"
BIN="vitals"
VERSION="latest"
BIN_DIR="${HOME}/.local/bin"
MODIFY_PATH=1
CHARSET=""                      # empty ⇒ leave config alone (charset auto ⇒ unicode)

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
	seed_config
	post_install
}

parse_args() {
	while [ $# -gt 0 ]; do
		case "$1" in
			--version) VERSION="$2"; shift 2 ;;
			--bin-dir) BIN_DIR="$2"; shift 2 ;;
			--charset) CHARSET="$2"; shift 2 ;;
			--nerdfont) CHARSET="nerdfont"; shift ;;
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

# seed_config writes a minimal config with the requested charset, but only when
# --charset/--nerdfont was passed AND no config exists yet — never clobbering an
# existing one. Icons (language glyphs, block/clock/branch) need charset nerdfont
# and a Nerd Font installed, so this stays strictly opt-in.
seed_config() {
	[ -n "$CHARSET" ] || return 0
	case "$CHARSET" in
		auto|unicode|nerdfont|ascii) ;;
		*) err "invalid --charset: ${CHARSET} (want auto|unicode|nerdfont|ascii)" ;;
	esac
	cfg_dir="${XDG_CONFIG_HOME:-${HOME}/.config}/vitals"
	cfg="${cfg_dir}/config.json"
	if [ -e "$cfg" ]; then
		say "Config exists (${cfg}); leaving it unchanged. Set \"charset\": \"${CHARSET}\" there to enable icons."
		return 0
	fi
	mkdir -p "$cfg_dir" || { say "warning: could not create ${cfg_dir}; skipping config seed"; return 0; }
	schema="https://raw.githubusercontent.com/${REPO}/main/schema/vitals.schema.json"
	cat > "$cfg" <<EOF
{
  "\$schema": "${schema}",
  "charset": "${CHARSET}"
}
EOF
	say "Wrote ${cfg} (charset: ${CHARSET})."
}

post_install() {
	case ":${PATH}:" in
		*":${BIN_DIR}:"*) ;;
		*) [ "$MODIFY_PATH" -eq 1 ] && say "Add to PATH:  export PATH=\"${BIN_DIR}:\$PATH\"" ;;
	esac
	cat <<EOF

Next steps:
  ${BIN} init                    # wire vitals into ~/.claude/settings.json (backup first)
  ${BIN} init --preset minimal   # ...and seed a starter config (full|minimal|compact)
  ${BIN} config                  # customize segments, options, theme, and order

Using a Nerd Font? Re-run with --nerdfont (or set "charset": "nerdfont" in
${XDG_CONFIG_HOME:-${HOME}/.config}/vitals/config.json) to show language and status icons.
EOF
}

need() { command -v "$1" >/dev/null 2>&1; }
say()  { printf '  %s\n' "$*"; }
err()  { printf 'error: %s\n' "$*" >&2; exit 1; }

main "$@"
