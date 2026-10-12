"""Apply the verified patch to the matching external wing source checkout."""
import hashlib
import pathlib
import shutil
import subprocess
import sys

BEFORE = '217c20a2c978f651139d18fd2c0436caab6d1e30bff3d1ce7f4048897f4b9188'
AFTER = 'c43b20e070d2eb8bd9fef3b98537e7d2746a475cceb9b25f70dfe1b549fb70e6'
KDAE = 'f7c1cb6b04364903a55dbf9790d6e5eca7d5a628'

def digest(file):
    return hashlib.sha256(file.read_bytes()).hexdigest()

def main():
    if len(sys.argv) != 2:
        raise SystemExit('Usage: python apply-wing-patch.py /absolute/path/to/wing')
    wing = pathlib.Path(sys.argv[1]).resolve(strict=True)
    source = wing / 'dae/run.go'
    current = digest(source)
    if current == AFTER:
        print('Verified patch already applied.')
        return
    if current != BEFORE:
        raise SystemExit('Source differs from the verified base; refusing to overwrite local changes.')
    core = subprocess.check_output(['git', '-C', str(wing / 'dae-core'), 'rev-parse', 'HEAD'], text=True).strip()
    if core != KDAE:
        raise SystemExit('kdae revision differs from the tested revision; port and revalidate the patch first.')
    patch = pathlib.Path(__file__).resolve().parent / 'patches/wing-reload-lifecycle.patch'
    subprocess.run(['git', 'apply', '--check', str(patch)], cwd=wing, check=True)
    backup = source.with_name('run.go.console-before')
    if backup.exists() and digest(backup) != BEFORE:
        raise SystemExit('Existing backup differs; preserve it before applying the patch.')
    if not backup.exists():
        shutil.copy2(source, backup)
    subprocess.run(['git', 'apply', str(patch)], cwd=wing, check=True)
    if digest(source) != AFTER:
        raise SystemExit('Patched hash differs from the tested source; inspect before building.')
    print('Applied verified lifecycle patch; original source preserved in dae/run.go.console-before.')

if __name__ == '__main__':
    main()
