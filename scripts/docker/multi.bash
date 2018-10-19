#!/usr/bin/env bash

set -xeuo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

. "$DIR/set_globals.bash"
"$DIR/clean.bash"
. "$DIR/ncpus.bash"

# Config
declare -r n="${n:-3}"
declare -r ip_start="${ip_start:-127.0.0.0}"
declare -r subnet="${subnet:-16}"
declare -r ip_range="$ip_start/$subnet"

# Install deps
"$DIR/install_deps.bash"

# Use -tags="netgo multi" in bgo build below to build multu lachesis version for testing
env GOOS=linux GOARCH=amd64 go build -tags="netgo multi" -ldflags "-linkmode external -extldflags -static -s -w" -o lachesis_linux cmd/lachesis/main.go || exit 1


# Create peers.json and lachesis_data_dir
batch-ethkey -dir "$BUILD_DIR/nodes" -network "$ip_start" -n "$n" > "$PEERS_DIR/peers.json"
rm -rf lachesis_data_dir
cp -r "$BUILD_DIR/nodes" lachesis_data_dir
cp "$PEERS_DIR/peers.json" lachesis_data_dir/

# Create loopback aliases and cp json.peers per node datadir
declare -i node_num=0
for ip in $(jq -rc '.[].NetAddr' "$PEERS_DIR/peers.json"); do
    cp "$PEERS_DIR/peers.json" lachesis_data_dir/$node_num/

    ip="${ip%:*}";
    echo $ip
    if [ "$ip" != "127.0.0.1" ]; then
	#	sudo route add -host $ip dev lo
	if  ( ifconfig lo:$node_num | grep -e 'inet 127\.' ) > /dev/null 2>& 1
	then
	    sudo ifconfig lo:$node_num down
	fi
	sudo ifconfig lo:$node_num $ip netmask 255.0.0.0 up
    else
	declare -r node_addr=$ip
    fi
  ((node_num++)) || true
  [ "$node_num" -gt "$n" ] && exit 0
done

# Run multi lachesis
GOMAXPROCS=$(($logicalCpuCount - 1)) ./lachesis_linux run --datadir ./lachesis_data_dir --store --listen=$node_addr:12000 --log=warn --heartbeat=4s -p $node_addr:9000 --test
