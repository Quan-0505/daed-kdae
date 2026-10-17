"""Apply narrowly scoped, hash-checked fixes to the pinned KDAE core."""
import hashlib,json,pathlib,shutil,subprocess,sys
base=pathlib.Path(__file__).resolve().parent
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def main():
 core=pathlib.Path(sys.argv[1]).resolve(strict=True)
 manifest=json.loads((base/'patches/core-manifest.json').read_text())
 patches=[]
 for name,item in manifest.items():
  current=sha(core/name)
  if current==item['after']:continue
  if current!=item['before']:raise SystemExit('Core source differs; refusing overwrite: '+name)
  patch=base/'patches'/item['patch']
  subprocess.run(['git','apply','--check',str(patch)],cwd=core,check=True)
  patches.append(patch)
 sources=list((base/'core-overlay').rglob('*.go'))
 for source in sources:
  target=core/source.relative_to(base/'core-overlay')
  if target.exists() and sha(target)!=sha(source):raise SystemExit('Core overlay differs: '+str(target))
 for patch in patches:subprocess.run(['git','apply',str(patch)],cwd=core,check=True)
 for source in sources:
  target=core/source.relative_to(base/'core-overlay');target.parent.mkdir(parents=True,exist_ok=True)
  shutil.copy2(source,target)
 for name,item in manifest.items():
  if sha(core/name)!=item['after']:raise SystemExit('Patched core hash differs: '+name)
 print('Verified embedded LPM batch compatibility fix installed.')
if __name__=='__main__':main()
