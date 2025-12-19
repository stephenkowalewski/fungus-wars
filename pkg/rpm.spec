%global debug_package %{nil}

%{!?_app_rpm_version: %define _app_rpm_version 0.0.0}
%{!?_app_version:     %define _app_version     unknown}
%{!?_app_git_commit:  %define _app_git_commit  unknown}
%{!?_app_git_status:  %define _app_git_status  unknown}

Name:           fungus-wars
Version:        %{_app_rpm_version}
Release:        1%{?dist}
Summary:        Fungus Wars web server

License:        MIT
URL:            https://github.com/stephenkowalewski/fungus-wars
Source0:        %{name}-%{version}.rpmsource.tar.gz

BuildRequires:  golang
BuildRequires:  make
BuildRequires:  systemd-rpm-macros

Requires(pre):  shadow-utils
Requires:       systemd

%description
Fungus Wars web server, with static assets and systemd service.

GIT_COMMIT=%{_app_git_commit}
GIT_STATUS=%{_app_git_status}

%prep
%autosetup -n %{name}-%{version}

%build
# Use SOURCE_DATE_EPOCH for reproducible builds
: "${SOURCE_DATE_EPOCH:=1700000000}"
BUILD_DATE="$(date -u -d "@$SOURCE_DATE_EPOCH" +'%Y-%m-%dT%H:%M:%SZ')"

export CGO_ENABLED=0
make build \
  STRIP=0 \
  VERSION="%{_app_version}" \
  DATE="$BUILD_DATE"

%check
export CGO_ENABLED=0
export GOPROXY=off
export GOSUMDB=off
go test ./...

%install
rm -rf %{buildroot}

install -d %{buildroot}%{_bindir}
install -m 0755 fungus-wars %{buildroot}%{_bindir}/fungus-wars

install -d %{buildroot}%{_datadir}/fungus-wars
cp -a static %{buildroot}%{_datadir}/fungus-wars/static

install -d %{buildroot}%{_unitdir}
install -m 0644 pkg/fungus-wars.service %{buildroot}%{_unitdir}/fungus-wars.service

%pre
# Create system group/user if missing
getent group fungus-wars >/dev/null || groupadd -r fungus-wars
getent passwd fungus-wars >/dev/null || \
  useradd -r -g fungus-wars -d / -s /sbin/nologin \
  -c "Fungus Wars service user" fungus-wars
exit 0

%post
%systemd_post fungus-wars.service

%preun
%systemd_preun fungus-wars.service

%postun
%systemd_postun_with_restart fungus-wars.service

# Remove user/group on erase only
if [ "$1" -eq 0 ]; then
  getent passwd fungus-wars >/dev/null && userdel fungus-wars || :
  getent group fungus-wars >/dev/null && groupdel fungus-wars || :
fi
exit 0

%files
%license LICENSE
%doc how_to_play.md

%caps(cap_net_bind_service=ep) %{_bindir}/fungus-wars
%{_datadir}/fungus-wars/static
%{_unitdir}/fungus-wars.service

%changelog
* Mon Dec 15 2025 Stephen Kowalewski - 0.9.0-1
- Initial build
