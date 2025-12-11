# scripts/build-deb.sh
#!/bin/bash
set -e

VERSION="1.0.0"
ARCH="amd64"
PACKAGE_NAME="hardware-collector"

# 构建二进制```bash
go build -o build/hardware-collector-client ./cmd/client
go build -o build/hardware-collector-cli ./cmd/cli

# 创建 DEB 包目录结构
DEB_DIR="build/deb/${PACKAGE_NAME}_${VERSION}_${ARCH}"
mkdir -p ${DEB_DIR}/DEBIAN
mkdir -p ${DEB_DIR}/usr/bin
mkdir -p ${DEB_DIR}/etc/hardware-collector
mkdir -p ${DEB_DIR}/etc/systemd/system
mkdir -p ${DEB_DIR}/var/log/hardware-collector
mkdir -p ${DEB_DIR}/var/cache/hardware-collector

# 复制文件
cp build/hardware-collector-client ${DEB_DIR}/usr/bin/
cp build/hardware-collector-cli ${DEB_DIR}/usr/bin/hardware-collector
cp configs/config.yaml ${DEB_DIR}/etc/hardware-collector/
cp systemd/hardware-collector.service ${DEB_DIR}/etc/systemd/system/

# 创建 control 文件
cat > ${DEB_DIR}/DEBIAN/control <<EOF
Package: ${PACKAGE_NAME}
Version: ${VERSION}
Architecture: ${ARCH}
Maintainer: Your Name <your@email.com>
Description: Hardware information collector
 Collects server hardware information including disk, memory, network, GPU
Depends: dmidecode
EOF

# 创建 postinst 脚本
cat > ${DEB_DIR}/DEBIAN/postinst <<EOF
#!/bin/bash
systemctl daemon-reload
systemctl enable hardware-collector.service
systemctl start hardware-collector.service
EOF
chmod +x ${DEB_DIR}/DEBIAN/postinst

# 构建 DEB 包
dpkg-deb --build ${DEB_DIR}
echo "DEB package created: ${DEB_DIR}.deb"