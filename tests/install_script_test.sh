#!/bin/sh

set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
PORTR_INSTALL_SH_LIB_ONLY=1 . "$ROOT/install.sh"

fail() {
    printf '%s\n' "$1" >&2
    exit 1
}

assert_eq() {
    expected="$1"
    actual="$2"
    message="$3"

    if [ "$expected" != "$actual" ]; then
        fail "$message: expected '$expected', got '$actual'"
    fi
}

assert_contains() {
    haystack="$1"
    needle="$2"
    message="$3"

    case "$haystack" in
        *"$needle"*) ;;
        *) fail "$message: expected to find '$needle' in '$haystack'" ;;
    esac
}

test_asset_version() {
    assert_eq "1.0.0" "$(asset_version "v1.0.0")" "asset_version strips leading v"
    assert_eq "1.0.0" "$(asset_version "1.0.0")" "asset_version keeps plain semver"
}

test_build_archive_name() {
    archive_name=$(build_archive_name "v1.0.0" "Darwin" "arm64")
    assert_eq "portr_1.0.0_Darwin_arm64.zip" "$archive_name" "build_archive_name uses asset version"
}

test_build_download_url() {
    download_url=$(build_download_url "v1.0.0" "Linux" "x86_64")
    assert_eq "https://github.com/amalshaji/portr/releases/download/v1.0.0/portr_1.0.0_Linux_x86_64.zip" "$download_url" "build_download_url keeps release tag and strips asset version"
}

test_download_file_uses_curl_fail_flag() {
    tmp_dir=$(mktemp -d)
    original_path=$PATH
    TEST_ARGS_FILE="$tmp_dir/args"
    export TEST_ARGS_FILE

    trap 'PATH="$original_path"; rm -rf "$tmp_dir"' EXIT HUP INT TERM

    cat > "$tmp_dir/curl" <<'EOF'
#!/bin/sh
while [ "$#" -gt 0 ]; do
    printf '%s\n' "$1" >> "$TEST_ARGS_FILE"
    if [ "$1" = "-o" ]; then
        shift
        printf '%s\n' "$1" >> "$TEST_ARGS_FILE"
        : > "$1"
    fi
    shift
done
EOF
    chmod +x "$tmp_dir/curl"

    PATH="$tmp_dir:$PATH"
    download_output="$tmp_dir/out"
    download_file "https://example.com/file.zip" "$download_output"

    curl_args=$(tr '\n' ' ' < "$TEST_ARGS_FILE")
    assert_contains "$curl_args" "-fsSL" "download_file uses curl fail flag"
    assert_contains "$curl_args" "https://example.com/file.zip" "download_file passes url"
    assert_contains "$curl_args" "$download_output" "download_file passes output path"

    PATH="$original_path"
    rm -rf "$tmp_dir"
    trap - EXIT HUP INT TERM
}

test_asset_version
test_build_archive_name
test_build_download_url
test_download_file_uses_curl_fail_flag
