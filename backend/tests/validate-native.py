"""Authenticated native API checks. --disposable permits clearing isolated test logs."""
import argparse,json,pathlib,urllib.request,urllib.error,time,threading
p=argparse.ArgumentParser();p.add_argument('--base',default='http://127.0.0.1:12070');p.add_argument('--session',default='/root/daed-console-test/live-session.json');p.add_argument('--output',default='/root/daed-console-test/native-report.json');p.add_argument('--disposable',action='store_true');args=p.parse_args()
token=json.loads(pathlib.Path(args.session).read_text())['token'];report={}
def call(path,data=None,method=None,auth=True):
 req=urllib.request.Request(args.base+'/api'+path,json.dumps(data).encode() if data is not None else None,{'Authorization':'Bearer '+token if auth else '', 'Content-Type':'application/json'},method=method)
 with urllib.request.urlopen(req,timeout=15) as response:return json.load(response)
try:call('/logs',auth=False);raise AssertionError('anonymous logs exposed')
except urllib.error.HTTPError as e:assert e.code==401
report['unauthenticatedDenied']=True
metrics=call('/runtime/metrics');assert metrics['rssBytes']>0 and metrics['heapBytes']>0 and metrics['goroutines']>0;time.sleep(1.1);metrics=call('/runtime/metrics');assert metrics['cpuPercent'] is not None and metrics['cpuPercent']>=0;report['metrics']=metrics
geo=call('/geodata');assert len(geo['items'])==2 and all(i['categories']>0 and i['entries']>0 for i in geo['items']);report['geodata']=[{k:v for k,v in item.items() if k!='source'} for item in geo['items']]
bundle=call('/config/bundle');assert bundle['format']=='daed-kdae' and 'users' not in bundle['tables'];preview=call('/config/bundle/preview',bundle);report['backupCounts']=preview['counts'];report['backupExcludesAccounts']=True
with urllib.request.urlopen(urllib.request.Request(args.base+'/api/config/dae',headers={'Authorization':'Bearer '+token}),timeout=15) as response:dae=response.read();assert b'global {' in dae and b'routing {' in dae
with urllib.request.urlopen(urllib.request.Request(args.base+'/api/config/dae?preview=true',dae,{'Authorization':'Bearer '+token,'Content-Type':'text/plain'}),timeout=15) as response:preview=json.load(response);assert preview['counts']['configs']==1
report['daeExportAndImportPreview']=True
settings=call('/logs/settings');assert settings['maxEntries']>=500;report['logLimits']=settings
original=call('/runtime/log-level')['level'];snapshot=call('/logs')
stream=urllib.request.urlopen(urllib.request.Request(args.base+'/api/events/logs?after_id='+str(snapshot['cursor'])+'&generation='+str(snapshot['generation']),headers={'Authorization':'Bearer '+token}),timeout=10)
call('/runtime/log-level',{'level':'warn'},'PATCH');seen=[]
def consume():
 for _ in range(100):
  line=stream.readline().decode()
  if 'Runtime log level updated' in line:seen.append(True);return
worker=threading.Thread(target=consume,daemon=True);worker.start();worker.join(8);stream.close();assert seen,'live SSE event missing'
filtered=call('/logs?level=warn&q=Runtime%20log%20level%20updated');assert filtered['items'];report['liveSSE']=True;report['keywordAndLevelFilter']=True
call('/runtime/log-level',{'level':original},'PATCH');report['runtimeLevelRestored']=True
if args.disposable:
 call('/logs',method='DELETE');assert not call('/logs')['items'];report['clearIsolatedLogs']=True
pathlib.Path(args.output).write_text(json.dumps(report,indent=2));print(json.dumps(report,indent=2))
