"""Package public release binaries/geodata only; never read a device data directory."""
import pathlib,sys,shutil,subprocess,tarfile,hashlib,json,time
out=pathlib.Path(sys.argv[1]).resolve();version=sys.argv[2]
assets=out/'assets';assets.mkdir(exist_ok=True)
unit='''[Unit]
Description=daed Go network console with kdae
Documentation=https://github.com/Quan-0505/daed-kdae
After=network-online.target systemd-sysctl.service
Wants=network-online.target
Conflicts=dae.service
[Service]
Type=simple
User=root
LimitNOFILE=1048576
ExecStart=/usr/bin/daed run -c /etc/daed/
Restart=on-failure
[Install]
WantedBy=multi-user.target
'''
init='''#!/bin/sh /etc/rc.common
START=95
STOP=10
USE_PROCD=1
start_service() {
 mkdir -p /etc/daed /tmp/log/daed
 procd_open_instance
 procd_set_param command /usr/bin/daed run -c /etc/daed/
 procd_set_param respawn
 procd_set_param limits nofile="1048576 1048576"
 procd_close_instance
}
'''
post='''#!/bin/sh
set -e
[ -n "${IPKG_INSTROOT:-}" ] && exit 0
mkdir -p /etc/daed /tmp/log/daed
if [ -x /etc/init.d/daed ]; then
 /etc/init.d/daed enable 2>/dev/null || true
 /etc/init.d/daed restart 2>/dev/null || true
fi
'''
pre='''#!/bin/sh
if [ -x /etc/init.d/daed ]; then
 /etc/init.d/daed stop 2>/dev/null || true
 /etc/init.d/daed disable 2>/dev/null || true
fi
'''
def put(root,name,data,mode=0o644):
 p=root/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_text(data,encoding='utf8');p.chmod(mode)
def payload(root,binary):
 p=root/'usr/bin/daed';p.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(out/'bin'/binary,p);p.chmod(0o755)
 for kind in ['geoip','geosite']:
  p=root/f'usr/share/daed/{kind}.dat';p.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(out/'geo'/f'{kind}.dat',p)
 # Explicit allowlist: packages must not contain account/configuration/backup files.
 allowed={'usr/bin/daed','usr/share/daed/geoip.dat','usr/share/daed/geosite.dat'}
 assert {p.relative_to(root).as_posix() for p in root.rglob('*') if p.is_file()}==allowed
for level,suffix in [('v2','v2_sse'),('v3','v3_avx2')]:
 root=out/f'deb-{level}';root.mkdir(exist_ok=False);payload(root,'daed-amd64-'+level)
 put(root,'usr/lib/systemd/system/daed.service',unit)
 put(root,'DEBIAN/control',f'''Package: daed
Version: {version}-kdae
Section: net
Priority: optional
Architecture: amd64
Depends: iproute2 (>= 6.7.0)
Maintainer: daed-kdae <noreply@github.com>
Description: Go daed console with pinned kdae engine
 Embedded management UI, live logs and scheduled geodata updates.
 Requires BTF and supported TC/eBPF kernel features; tested on Linux 6.12.
''')
 put(root,'DEBIAN/postinst','''#!/bin/sh
set -e
mkdir -p /etc/daed
if command -v systemctl >/dev/null 2>&1; then
 systemctl daemon-reload
 systemctl enable daed.service >/dev/null 2>&1 || true
 if [ -d /run/systemd/system ]; then systemctl restart daed.service || true; fi
fi
''',0o755)
 subprocess.run(['dpkg-deb','--root-owner-group','--build',str(root),str(assets/f'daed-kdae_{version}-linux-x86_64_{suffix}.deb')],check=True)
for platform in ['r2s','r3s','r4s','x86']:
 root=out/f'apk-{platform}';root.mkdir(exist_ok=False);payload(root,'daed-amd64-v2' if platform=='x86' else 'daed-arm64')
 put(root,'etc/init.d/daed',init,0o755)
 size=sum(p.stat().st_size for p in root.rglob('*') if p.is_file())
 # This is the legacy v2 tar payload used by the existing SDK conversion workflow.
 put(root,'.PKGINFO',f'''pkgname = daed-kdae
pkgver = {version}-r1
pkgdesc = Go daed/kdae console with live logs and scheduled updates
url = https://github.com/Quan-0505/daed-kdae
arch = {'x86_64' if platform=='x86' else 'aarch64_generic'}
size = {size}
license = AGPL-3.0-only
''')
 put(root,'.post-install',post,0o755);put(root,'.post-upgrade',post,0o755);put(root,'.pre-deinstall',pre,0o755)
 with tarfile.open(assets/f'daed-kdae-{platform}.apk','w:gz',format=tarfile.PAX_FORMAT) as t:
  for p in sorted(root.rglob('*')):
   if p.is_file():
    info=t.gettarinfo(str(p),arcname=p.relative_to(root).as_posix());info.uid=info.gid=0;info.uname=info.gname='root'
    with p.open('rb') as f:t.addfile(info,f)
# Verify Debian and legacy APK payloads against the compiled binary hashes.
for p in assets.glob('*.deb'):
 dest=out/('verify-'+p.stem);subprocess.run(['dpkg-deb','-x',str(p),str(dest)],check=True)
 level='v3' if 'v3_avx2' in p.name else 'v2';assert (dest/'usr/bin/daed').read_bytes()==(out/'bin'/f'daed-amd64-{level}').read_bytes()
for p in assets.glob('*.apk'):
 with tarfile.open(p) as t:
  names=t.getnames();assert not any(n.endswith(('.db','.sqlite','.json','.log')) for n in names)
  assert t.extractfile('usr/bin/daed').read()==(out/'bin'/('daed-amd64-v2' if 'x86' in p.name else 'daed-arm64')).read_bytes()
print('PASS: six packages, payload hashes match, no device database/configuration files.')
