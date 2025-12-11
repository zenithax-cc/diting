# scripts/build-rpm.sh
#!/bin/bash
set -e

VERSION="1.0.0"
PACKAGE_NAME="hardware-collector"

# 构建二进制
go build -o build/hardware-collector-client ./cmd/client
go build -o build/hardware-collector-cli ./cmd/cli

# 创建 RPM 构建目录
mkdir -p ~/rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# 创建 spec 文件
cat > ~/rpmbuild/SPECS/${PACKAGE_NAME}.spec <<EOF
Name:           ${PACKAGE_NAME}
Version:        ${VERSION}
Release:        1%{?dist}
Summary:        Hardware information collector
License:        MIT
Requires:       dmidecode

%description
Collects server hardware information including disk, memory, network, GPU

%install
mkdir -p %{buildroot}/usr/bin
mkdir -p %{buildroot}/etc/hardware-collector
mkdir -p %{buildroot}/etc/systemd/system
install -m 0755 ${PWD}/build/hardware-collector-client %{buildroot}/usr/bin/
install -m 0755 ${PWD}/build/hardware-collector-cli %{buildroot}/usr/bin/hardware-collector
install -m 0644 ${PWD}/configs/config.yaml %{buildroot}/etc/hardware-collector/
install -m 0644 ${PWD}/systemd/hardware-collector.service %{buildroot}/etc/systemd/system/

%post
systemctl daemon-reload
systemctl enable hardware-collector.service
systemctl start hardware-collector.service

%files
/usr/bin/hardware-collector-client
/usr/bin/hardware-collector
/etc/hardware-collector/config.yaml
/etc/systemd/system/hardware-collector.service

%changelog
* $(date +"%a %b %d %Y") Developer <dev@example.com> - ${VERSION}-1
- Initial package
EOF

# 构建 RPM
rpmbuild -ba ~/rpmbuild/SPECS/${PACKAGE_NAME}.spec
echo "RPM package created in ~/rpmbuild/RPMS/"