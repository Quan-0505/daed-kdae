#!/usr/bin/env bash
set -euo pipefail
# Usage: build-release.sh /absolute/wing /absolute/production-dist /absolute/output [2.2.5]
wing=$(realpath "$1")
web=$(realpath "$2")
out=$(realpath -m "$3")
version=${4:-2.2.5}
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Invalid version'; exit 1; }
repo=$(cd "$(dirname "$0")/.." && pwd)
mkdir -p "$out/bin" "$out/geo"
python3 "$repo/backend/prepare-wing.py" "$wing"
cd "$wing"
make deps
go test -race ./dae ./maintenance ./observability ./consoleapi ./graphql ./graphql/service/...
CGO_ENABLED=0 make OUTPUT="$out/bin/daed-amd64-v2" APPNAME=daed WEB_DIST="$web" VERSION="v$version-kdae" bundle
ldf="-s -w -X github.com/daeuniverse/dae-wing/db.AppVersion=v$version-kdae -X github.com/daeuniverse/dae-wing/db.AppName=daed"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v2 go build -tags=embedallowed -trimpath -ldflags="$ldf" -o "$out/bin/daed-amd64-v2" .
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v3 go build -tags=embedallowed -trimpath -ldflags="$ldf" -o "$out/bin/daed-amd64-v3" .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags=embedallowed -trimpath -ldflags="$ldf" -o "$out/bin/daed-arm64" .
for kind in geosite geoip; do
 source="https://cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/$kind.dat"
 fallback="https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/$kind.dat"
 curl -fsSL --connect-timeout 15 --max-time 120 --retry 2 "$source" -o "$out/geo/$kind.dat.part" || \
   curl -fsSL --connect-timeout 15 --max-time 120 --retry 2 "$fallback" -o "$out/geo/$kind.dat.part"
 mv "$out/geo/$kind.dat.part" "$out/geo/$kind.dat"
done
python3 "$repo/packaging/package-release.py" "$out" "$version"
file "$out"/bin/*
"$out/bin/daed-amd64-v2" --version
