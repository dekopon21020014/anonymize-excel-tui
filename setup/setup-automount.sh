#!/bin/bash
set -e
mkdir -p /media/cdrom
cp ./media-cdrom.mount /etc/systemd/system/media-cdrom.mount
cp ./media-cdrom.automount /etc/systemd/system/media-cdrom.automount

# cd/dvdをマウントするための設定
systemctl daemon-reload
systemctl enable media-cdrom.mount
systemctl start media-cdrom.automount

# デスクトップにショートカットを作成
cp cdrom-mount.desktop ~/Desktop/
echo "正常に終了しました"
