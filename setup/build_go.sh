#!/bin/bash

set -e  # エラーが発生したら即終了

GO_TAR="./go1.22.1.linux-amd64.tar.gz"
INSTALL_DIR="/usr/local"
GO_DIR="${INSTALL_DIR}/go"
PROFILE_FILE="$HOME/.bashrc"

# 1. 既存の Go を削除
if [ -d "$GO_DIR" ]; then
    echo "既存の Go を削除します..."
    sudo rm -rf "$GO_DIR"
fi

# 2. tar.gz を展開
echo "Go をインストール中..."
sudo tar -C "$INSTALL_DIR" -xzf "$GO_TAR"

# 3. PATH に追加（.bashrc に追記）
if ! grep -q 'export PATH=$PATH:/usr/local/go/bin' "$PROFILE_FILE"; then
    echo 'export PATH=$PATH:/usr/local/go/bin' >> "$PROFILE_FILE"
    echo "環境変数を設定しました。"
fi

# 4. 設定を適用
source "$PROFILE_FILE"

# 5. インストール確認
go version && echo "Go のインストールが完了しました！"
