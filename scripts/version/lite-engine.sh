
#!/bin/bash

# Script to check version consistency between go.mod and config.go

check_versions() {
    # Extract version from go.mod
    GO_MOD_VERSION=$(grep "github.com/harness/lite-engine" go.mod | awk '{print $2}')

    # Extract version from config.go
    CONFIG_VERSION=$(grep "VM_BINARY_URI_LITE_ENGINE.*default" delegateshell/delegate/config.go | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+')

    if [ "$GO_MOD_VERSION" != "$CONFIG_VERSION" ]; then
        echo "❌ Version mismatch detected!"
        echo "go.mod version: $GO_MOD_VERSION"
        echo "config.go version: $CONFIG_VERSION"
        exit 1
    else
        echo "✅ Versions match: $GO_MOD_VERSION"
    fi
}

update_versions() {
    # Extract version from go.mod
    GO_MOD_VERSION=$(grep "github.com/harness/lite-engine" go.mod | awk '{print $2}')

    # Update version in config.go
    sed -i.bak "s/v[0-9]\+\.[0-9]\+\.[0-9]\+/$GO_MOD_VERSION/" delegate/config.go
    rm -f delegate/config.go.bak

    echo "Updated config.go to version $GO_MOD_VERSION"
}

case "$1" in
    "check")
        check_versions
        ;;
    "update")
        update_versions
        ;;
    *)
        echo "Usage: $0 {check|update}"
        exit 1
        ;;
esac
