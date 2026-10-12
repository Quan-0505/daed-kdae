import json,os,pathlib,signal,subprocess,time,urllib.request
root=pathlib.Path('/root/daed-console-test');old_pids=[]
for process_path in pathlib.Path('/proc').iterdir():
 if not process_path.name.isdigit():continue
 try:command=(process_path/'cmdline').read_bytes()
 except (FileNotFoundError,PermissionError):continue
 if command.startswith(str(root/'bin/dae-wing').encode()) and (str(root/'state').encode()+b'\x00') in command:old_pids.append(int(process_path.name))
for old in old_pids:os.kill(old,signal.SIGTERM)
time.sleep(2)
for old in old_pids:
 try:os.kill(old,signal.SIGKILL)
 except ProcessLookupError:pass
time.sleep(.5)
log=open(root/'runtime.log','w');process=subprocess.Popen([str(root/'bin/dae-wing'),'run','-c',str(root/'state'),'-l','127.0.0.1:12070','--disable-timestamp'],stdout=log,stderr=subprocess.STDOUT,start_new_session=True);(root/'test.pid').write_text(str(process.pid))
session=json.loads((root/'live-session.json').read_text());session['pid']=process.pid;(root/'live-session.json').write_text(json.dumps(session))
def request(query):
 req=urllib.request.Request('http://127.0.0.1:12070/graphql',json.dumps({'query':query}).encode(),{'Content-Type':'application/json','Authorization':'Bearer '+session['token']});result=json.load(urllib.request.urlopen(req,timeout=30));assert not result.get('errors'),result;return result['data']
for _ in range(50):
 if process.poll() is not None:raise RuntimeError((root/'runtime.log').read_text())
 try:request('query { healthCheck }');break
 except Exception:time.sleep(.2)
print('Restart:',request('mutation { run(dry:false) }'));print('Status:',request('query { healthCheck general { dae { running } } }'))
