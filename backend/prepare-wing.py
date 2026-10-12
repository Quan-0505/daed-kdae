"""Prepare a pinned, clean dae-wing checkout for the sole Go console build."""
import argparse,hashlib,json,pathlib,subprocess,sys
WING='dc503088945812c11235b35362d2bfa1a4c3bdf0'
CORE='f7c1cb6b04364903a55dbf9790d6e5eca7d5a628'
base=pathlib.Path(__file__).resolve().parent

def run(*args,cwd=None):subprocess.run(args,cwd=cwd,check=True)
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def main():
 parser=argparse.ArgumentParser(description=__doc__)
 parser.add_argument('wing',type=pathlib.Path)
 parser.add_argument('--clone',action='store_true',help='Clone pinned upstream sources into a new directory')
 args=parser.parse_args();wing=args.wing.resolve()
 if args.clone:
  if wing.exists():raise SystemExit('Clone destination already exists; refusing to overwrite.')
  run('git','clone','https://github.com/daeuniverse/dae-wing.git',str(wing))
  run('git','checkout','--detach',WING,cwd=wing)
  run('git','clone','https://github.com/olicesx/dae.git',str(wing/'dae-core'))
  run('git','checkout','--detach',CORE,cwd=wing/'dae-core')
 for p,revision in [(wing,WING),(wing/'dae-core',CORE)]:
  if subprocess.check_output(['git','rev-parse','HEAD'],cwd=p,text=True).strip()!=revision:raise SystemExit('Pinned source revision differs: '+str(p))
 run('git','submodule','update','--init','--recursive',cwd=wing/'dae-core')
 manifest=json.loads((base/'patches/bootstrap-manifest.json').read_text())
 final=json.loads((base/'patches/console-manifest.json').read_text())
 patches=[]
 for name,item in manifest.items():
  current=sha(wing/name)
  if current==item['after'] or current==final.get(name,{}).get('after'):continue
  if current!=item['before']:raise SystemExit('Unknown source changes; refusing to overwrite: '+name)
  patch=base/'patches'/item['patch'];run('git','apply','--check',str(patch),cwd=wing);patches.append(patch)
 for patch in patches:run('git','apply',str(patch),cwd=wing)
 run(sys.executable,str(base/'apply-console-features.py'),str(wing))
 print('Pinned wing/kdae source prepared. Build only with the production dist/.')
if __name__=='__main__':main()
