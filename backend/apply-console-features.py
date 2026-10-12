"""Install verified console additions on matching external wing revisions."""
import hashlib,json,pathlib,shutil,subprocess,sys
base=pathlib.Path(__file__).resolve().parent
def digest(file):return hashlib.sha256(file.read_bytes()).hexdigest()
def main():
 if len(sys.argv)!=2:raise SystemExit('Usage: python3 apply-console-features.py /absolute/path/to/wing')
 wing=pathlib.Path(sys.argv[1]).resolve(strict=True)
 manifest=json.loads((base/'patches/console-manifest.json').read_text())
 transitions=json.loads((base/'patches/console-transitions.json').read_text())
 core=subprocess.check_output(['git','-C',str(wing/'dae-core'),'rev-parse','HEAD'],text=True).strip()
 if core!='f7c1cb6b04364903a55dbf9790d6e5eca7d5a628':raise SystemExit('kdae revision differs; port and test first.')
 if digest(wing/'dae/run.go')=='217c20a2c978f651139d18fd2c0436caab6d1e30bff3d1ce7f4048897f4b9188':
  subprocess.run([sys.executable,str(base/'apply-wing-patch.py'),str(wing)],check=True)
 patches=[]
 for name,value in manifest.items():
  current=digest(wing/name)
  if current==value['after']:continue
  patch=transitions.get(name,{}).get(current)
  if patch is None:raise SystemExit('Source differs from verified revisions; refusing to overwrite: '+name)
  patches.append((name,current,base/'patches'/patch))
 sources=list((base/'overlay').rglob('*.go'))
 upgrades=json.loads((base/'patches/overlay-upgrades.json').read_text())
 for source in sources:
  name=source.relative_to(base/'overlay').as_posix();target=wing/name
  if target.exists() and digest(target)!=digest(source):
   known=upgrades.get(name,{})
   if digest(target) not in known.get('before',[]) or digest(source)!=known.get('after'):raise SystemExit('Existing overlay differs: '+str(target))
 for name,current,patch in patches:
  subprocess.run(['git','apply','--check',str(patch)],cwd=wing,check=True)
 for name,current,patch in patches:
  source=wing/name;backup=source.with_name(source.name+'.console-before-'+current[:12])
  if backup.exists() and digest(backup)!=current:raise SystemExit('Backup differs: '+str(backup))
  if not backup.exists():shutil.copy2(source,backup)
  subprocess.run(['git','apply',str(patch)],cwd=wing,check=True)
 for source in sources:
  target=wing/source.relative_to(base/'overlay');target.parent.mkdir(parents=True,exist_ok=True)
  if target.exists() and digest(target)!=digest(source):
   current=digest(target);backup=target.with_name(target.name+'.console-before-'+current[:12])
   if backup.exists() and digest(backup)!=current:raise SystemExit('Overlay backup differs: '+str(backup))
   if not backup.exists():shutil.copy2(target,backup)
   shutil.copy2(source,target)
  elif not target.exists():shutil.copy2(source,target)
 for name,value in manifest.items():
  if digest(wing/name)!=value['after']:raise SystemExit('Patched source hash differs: '+name)
 print('Verified console features installed. Run go test -race ./maintenance ./observability ./consoleapi ./graphql ./graphql/service/subscription, then rebuild with WEB_DIST.')
if __name__=='__main__':main()
