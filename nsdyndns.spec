%bcond_without check

Name:           nsdyndns
Version:        0.1.0
# Release: computed by rpm-build.sh (--define release N = commits since
# the last tag + 1); defaults to 1 when release is not defined.
Release:        %{?release}%{!?release:1}%{?dist}
Summary:        Dynamic DNS update daemon for GoDaddy and DigitalOcean
# Cumulative SPDX expression (project AND union of all vendored module
# licenses); verified by rpm-build.sh (license_expression) against the
# LICENSE files of the project and the vendored modules.
License:        Apache-2.0 AND BSD-3-Clause
URL:            https://github.com/simon3z/nsdyndns
# Source0: git archive of the project
# Source1: vendored Go modules (regenerated at build time from go.mod/go.sum)
Source0:        %{name}-%{version}.tar.xz
Source1:        %{name}-%{version}-vendor.tar.xz

ExclusiveArch:  x86_64 aarch64
BuildRequires:  golang

%description
nsdyndnsd is a dynamic DNS update daemon. It periodically determines the
external IP address of the host and updates the A record of the configured
domain with the GoDaddy and/or DigitalOcean API, whenever the address has
changed.

%prep
# -a 1: extract the vendor tarball (Source1) on top of Source0
%autosetup -a 1
# System user and group for the daemon, defined declaratively so rpm
# (>= 4.19) creates them from this file during installation.
cat > nsdyndns-sysusers.conf <<'EOF'
g nsdyndns -
u nsdyndns - - /var/lib/nsdyndns /usr/sbin/nologin
EOF

%build
# GOCACHE must be writable; the build is fully offline thanks to the
# committed vendor/ tree (go uses -mod=vendor automatically).
export GOCACHE=%{_builddir}/gocache
CGO_ENABLED=0 go build -mod=vendor -o nsdyndnsd ./cmd/nsdyndnsd

%install
install -Dm 0755 nsdyndnsd %{buildroot}%{_bindir}/nsdyndnsd
# Ship the example config where the unit file expects it (/etc).
install -Dm 0644 init/nsdyndnsd.conf %{buildroot}/etc/nsdyndnsd.conf
install -Dm 0644 init/nsdyndnsd.service %{buildroot}/usr/lib/systemd/system/nsdyndnsd.service
# rpm's sysusers handler creates the user but not its home directory
# (useradd without -m), so package it for the unit's WorkingDirectory.
install -d %{buildroot}/var/lib/nsdyndns
# Ship the sysusers config defining the nsdyndns user and group; rpm
# creates them from this file during installation, so the unit's
# User=/Group= and WorkingDirectory exist before the service can start.
install -Dm 0644 nsdyndns-sysusers.conf %{buildroot}%{_sysusersdir}/%{name}.conf

%check
%if %{with check}
export GOCACHE=%{_builddir}/gocache
go test -mod=vendor ./...
%endif

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun %{name}.service

%files
%license LICENSE
%doc README.md
%{_bindir}/nsdyndnsd
%config(noreplace) /etc/nsdyndnsd.conf
/usr/lib/systemd/system/nsdyndnsd.service
%dir /var/lib/nsdyndns
%{_sysusersdir}/%{name}.conf

%changelog
* Wed Oct 14 2026 Federico Simoncelli <federico.simoncelli@gmail.com> - 0.1.0-%{?release}%{!?release:1}
- Initial package release
