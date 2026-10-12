import json,pathlib,subprocess,sqlite3,time,urllib.request,signal,sys,re
root=pathlib.Path('/root/daed-console-test')
saved={};names=[];processes=[];firewall=[];report={};test_node=None;test_group=None;daemon_started=False
db=sqlite3.connect(root/'state/wing.db')
original=db.execute('select routing from routings where selected=1').fetchone()[0]
original_global=db.execute('select global from configs where selected=1').fetchone()[0]
for role,expected in [('wan','daetest0'),('lan','daetest1')]:
 if not re.search(r'(?m)^\s*'+role+r'_interface:\s*'+expected+r'\s*$',original_global):raise RuntimeError('Test DB must select only the isolated '+role+' interface '+expected)
session=json.loads((root/'live-session.json').read_text())
def run(*args):return subprocess.check_output(args,text=True).strip()
def gql(query):
 req=urllib.request.Request('http://127.0.0.1:12070/graphql',json.dumps({'query':query}).encode(),{'Content-Type':'application/json','Authorization':'Bearer '+session['token']})
 result=json.load(urllib.request.urlopen(req,timeout=30))
 if result.get('errors'):raise RuntimeError(result['errors'])
 return result['data']
def routing(value):
 db.execute('update routings set routing=?,version=version+1 where selected=1',(value,));db.commit()
 assert gql('mutation { run(dry:false) }')['run']==1
 time.sleep(1)
for reserved in ['daetest0','daetest1','dae0']:
 if subprocess.run(['ip','link','show',reserved],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode==0:raise RuntimeError('Reserved test interface already exists: '+reserved)
if any(ns in run('ip','netns','list') for ns in ['console-client','console-server','daens']):raise RuntimeError('Reserved test namespace already exists')
server='''import socket,threading,time
def tcp():
 s=socket.socket();s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.bind(('0.0.0.0',18080));s.listen()
 while True:
  c,a=s.accept()
  try:
   request=c.recv(4096);c.sendall(b'HTTP/1.1 200 OK\\r\\nContent-Length: 1048576\\r\\nConnection: close\\r\\n\\r\\n')
   for _ in range(64):
    c.sendall(b'x'*16384)
    if b'/slow ' in request:time.sleep(.08)
  except OSError:pass
  c.close()
threading.Thread(target=tcp,daemon=True).start()
def socks_conn(c):
 import select,struct
 def exact(n):
  d=b''
  while len(d)<n:
   chunk=c.recv(n-len(d))
   if not chunk:raise EOFError()
   d+=chunk
  return d
 try:
  h=exact(2);exact(h[1]);c.sendall(b'\\x05\\x00');h=exact(4)
  assert h[1] in (1,3),h
  if h[3]==1:host=socket.inet_ntoa(exact(4))
  elif h[3]==3:host=exact(exact(1)[0]).decode()
  else:raise AssertionError(h)
  port=struct.unpack('!H',exact(2))[0]
  if h[1]==3:
   u=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);u.bind(('10.99.10.2',0));u.settimeout(.2)
   c.sendall(b'\\x05\\x00\\x00\\x01'+socket.inet_aton('10.99.10.2')+struct.pack('!H',u.getsockname()[1]))
   stop=threading.Event()
   def udp_relay():
    while not stop.is_set():
     try:
      packet,peer=u.recvfrom(65535);assert packet[:4]==b'\\x00\\x00\\x00\\x01'
      destination=(socket.inet_ntoa(packet[4:8]),struct.unpack('!H',packet[8:10])[0])
      outbound=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);outbound.settimeout(2);outbound.sendto(packet[10:],destination);reply=outbound.recv(65535);outbound.close();u.sendto(packet[:10]+reply,peer)
      print('SOCKS UDP',len(reply),flush=True)
     except socket.timeout:continue
     except (OSError,AssertionError):break
   worker=threading.Thread(target=udp_relay,daemon=True);worker.start()
   try:
    while c.recv(1024):pass
   finally:stop.set();worker.join(timeout=1);u.close()
   return
  print('SOCKS CONNECT',host,port,flush=True)
  remote=socket.create_connection((host,port),3);c.sendall(b'\\x05\\x00\\x00\\x01'+socket.inet_aton('10.99.10.2')+struct.pack('!H',1080))
  while True:
   readable,_,_=select.select([c,remote],[],[],5)
   if not readable:break
   for source in readable:
    d=source.recv(65536)
    if not d:return
    (remote if source is c else c).sendall(d)
 except (OSError,EOFError,AssertionError) as error:print('SOCKS ERROR',repr(error),flush=True)
 finally:
  c.close()
  if 'remote' in locals():remote.close()
def socks():
 s=socket.socket();s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.bind(('0.0.0.0',1080));s.listen()
 while True:
  c,a=s.accept();threading.Thread(target=socks_conn,args=(c,),daemon=True).start()
threading.Thread(target=socks,daemon=True).start()
s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);s.bind(('0.0.0.0',18081))
while True:
 d,a=s.recvfrom(65535);s.sendto(d,a)
'''
client='''import socket,json
s=socket.create_connection(('10.99.10.2',18080),3);s.settimeout(3);s.sendall(b'GET / HTTP/1.1\\r\\nHost: isolated-test\\r\\n\\r\\n');data=b''
while True:
 d=s.recv(65536)
 if not d:break
 data+=d
assert len(data.split(b'\\r\\n\\r\\n',1)[1])==1048576
s.close();s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);s.settimeout(3)
for i in range(100):
 payload=(str(i)+':').encode()+b'u'*1000;s.sendto(payload,('10.99.10.2',18081));assert s.recv(2048)==payload
print(json.dumps({'tcpBodyBytes':1048576,'udpEchoPackets':100}))
'''
try:
 existing=run('ip','netns','list')
 for ns,iface,peer,network in [('console-client','daetest1','ctclient','10.99.11'),('console-server','daetest0','ctserver','10.99.10')]:
  if ns in existing:raise RuntimeError('Namespace occupied: '+ns)
  if subprocess.run(['ip','link','show',iface],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode==0:raise RuntimeError('Interface occupied: '+iface)
  run('ip','netns','add',ns);names.append(ns)
  run('ip','link','add',iface,'type','veth','peer','name',peer)
  run('ip','link','set',peer,'netns',ns);run('ip','addr','add',network+'.1/24','dev',iface);run('ip','link','set',iface,'up')
  run('ip','-n',ns,'addr','add',network+'.2/24','dev',peer);run('ip','-n',ns,'link','set',peer,'up');run('ip','-n',ns,'link','set','lo','up');run('ip','-n',ns,'route','add','default','via',network+'.1')
 for iface in ['all','default','daetest0','daetest1']:
  for setting in ['send_redirects','accept_redirects','secure_redirects','accept_source_route','rp_filter']:
   key=f'net.ipv4.conf.{iface}.{setting}';saved[key]=run('sysctl','-n',key);run('sysctl','-qw',key+'=0')
 for key in ['net.ipv4.conf.all.forwarding','net.ipv6.conf.all.forwarding']:
  saved[key]=run('sysctl','-n',key);run('sysctl','-qw',key+'=1')
 test_global=original_global.replace('log_level: info','log_level: debug').rsplit('}',1)[0]+'\ndial_mode: ip\ntcp_check_url: "http://10.99.10.2:18080"\ntcp_check_http_method: GET\ncheck_interval: 1s\n}\n'
 db.execute('update configs set global=? where selected=1',(test_global,));db.commit()
 for src,dst in [('daetest1','daetest0'),('daetest0','daetest1'),('daetest1','dae0'),('dae0','daetest1'),('dae0','daetest0'),('daetest0','dae0')]:
  rule=['FORWARD','-i',src,'-o',dst,'-m','comment','--comment','codex-console-isolated-test','-j','ACCEPT']
  run('iptables','-I',*rule);firewall.append(rule)
 processes.append(subprocess.Popen(['ip','netns','exec','console-server','python3','-u','-c',server],stdout=open(root/'traffic-server.log','w'),stderr=subprocess.STDOUT))
 daemon_started=True
 run('python3',str(root/'restart-final.py'))
 time.sleep(1)
 query='query { healthCheck general { runtimeOverview(windowSec:60,maxPoints:120) { updatedAt uploadTotal downloadTotal activeConnections udpSessions samples { timestamp uploadRate downloadRate } } } }'
 report['before']=gql(query)
 report['direct']=json.loads(run('ip','netns','exec','console-client','python3','-c',client))
 time.sleep(2);report['after']=gql(query)
 routing('routing {\nfallback: block\n}\n')
 blocked=subprocess.run(['ip','netns','exec','console-client','python3','-c',client],capture_output=True,text=True,timeout=12)
 report['blockControl']={'blocked':blocked.returncode!=0,'exitCode':blocked.returncode}
 assert blocked.returncode!=0,'Traffic bypassed kdae block rule'
 routing(original)
 report['afterReload']=json.loads(run('ip','netns','exec','console-client','python3','-c',client))
 test_node=db.execute("insert into nodes(link,name,address,protocol) values(?,?,?,?)",('socks5://10.99.10.2:1080','console-test-socks','10.99.10.2','socks5')).lastrowid
 test_group=db.execute("insert into groups(name,policy,version) values(?,?,0)",('console_test_proxy','fixed')).lastrowid
 db.execute('insert into group_policy_params(key,value,group_id) values(?,?,?)',('','0',test_group))
 db.execute('insert into group_nodes(group_id,node_id) values(?,?)',(test_group,test_node));db.commit()
 proxy_routing='routing {\ndport(18080,18081) -> console_test_proxy\nfallback: direct\n}\n'
 routing(proxy_routing)
 time.sleep(3)
 for ns in ['daens']:
  for key in ['net.ipv4.conf.all.rp_filter','net.ipv4.conf.default.rp_filter','net.ipv4.conf.dae0peer.rp_filter']:
   subprocess.run(['ip','netns','exec',ns,'sysctl','-qw',key+'=0'],stderr=subprocess.DEVNULL)
 capture=open(root/'traffic-capture.log','w')
 processes.append(subprocess.Popen(['tcpdump','-n','-l','-i','any','tcp port 1080 or tcp port 18080'],stdout=capture,stderr=subprocess.STDOUT))
 processes.append(subprocess.Popen(['ip','netns','exec','daens','tcpdump','-n','-l','-i','any','tcp port 1080 or tcp port 18080'],stdout=capture,stderr=subprocess.STDOUT))
 time.sleep(.3)
 report['proxyTransfer']=json.loads(run('ip','netns','exec','console-client','python3','-c',client))
 time.sleep(2);report['proxyStats']=gql(query)
 stats=report['proxyStats']['general']['runtimeOverview']
 assert int(stats['downloadTotal'])>=1048576,stats
 assert int(stats['uploadTotal'])>0,stats
 report['proxyReloads']=[]
 for i in range(3):
  routing(proxy_routing)
  result=json.loads(run('ip','netns','exec','console-client','python3','-c',client))
  report['proxyReloads'].append({'reload':i+1,'transfer':result,'health':gql('query { healthCheck }')['healthCheck']})
 stream_marker=root/'stream-started';stream_marker.unlink(missing_ok=True)
 stream_client=client.replace('GET / HTTP','GET /slow HTTP').replace("data=b''",f"data=b'';open({str(stream_marker)!r},'w').write('started')")
 stream_log=open(root/'stream-result.json','w')
 stream=subprocess.Popen(['ip','netns','exec','console-client','python3','-c',stream_client],stdout=stream_log,stderr=open(root/'stream-error.log','w'));processes.append(stream)
 for _ in range(50):
  if stream_marker.exists():break
  time.sleep(.1)
 assert stream_marker.exists(),'Long connection never started'
 time.sleep(.3)
 report['streamBeforeReload']=gql(query)
 routing(proxy_routing)
 assert stream.wait(timeout=15)==0,(root/'stream-error.log').read_text()
 stream_log.close();report['longConnection']=json.loads((root/'stream-result.json').read_text())
 db.execute('delete from group_policy_params where group_id=?',(test_group,));db.commit()
 invalid=urllib.request.Request('http://127.0.0.1:12070/graphql',json.dumps({'query':'mutation { run(dry:false) }'}).encode(),{'Content-Type':'application/json','Authorization':'Bearer '+session['token']})
 invalid_result=json.load(urllib.request.urlopen(invalid,timeout=15))
 assert invalid_result.get('errors'),'Invalid fixed policy was not rejected'
 report['invalidCandidate']={'rejected':True,'message':invalid_result['errors'][0]['message'],'health':gql('query { healthCheck }')['healthCheck'],'oldTransfer':json.loads(run('ip','netns','exec','console-client','python3','-c',client))}
 db.execute('insert into group_policy_params(key,value,group_id) values(?,?,?)',('','0',test_group));db.commit()
 routing(proxy_routing)
 report['recovery']=json.loads(run('ip','netns','exec','console-client','python3','-c',client))
 changed_port=test_global.replace('tproxy_port: 12345','tproxy_port: 23456')
 db.execute('update configs set global=? where selected=1',(changed_port,));db.commit();routing(proxy_routing)
 report['portChange']=json.loads(run('ip','netns','exec','console-client','python3','-c',client))
 db.execute('update configs set global=? where selected=1',(test_global,));db.commit();routing(proxy_routing)
 report['portRestore']=json.loads(run('ip','netns','exec','console-client','python3','-c',client))
 if '--hold' in sys.argv:
  (root/'ui-ready.json').write_text(json.dumps({'ready':True,'time':time.time()}))
  print('UI validation ready',flush=True)
  deadline=time.time()+120
  while time.time()<deadline:
   if not gql('query { general { dae { running } } }')['general']['dae']['running']:break
   transfer=subprocess.run(['ip','netns','exec','console-client','python3','-c',stream_client],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
   if transfer.returncode and not gql('query { general { dae { running } } }')['general']['dae']['running']:break
   assert transfer.returncode==0,'Traffic failed during browser validation'
  (root/'ui-ready.json').unlink(missing_ok=True)
 report['health']=gql('query { healthCheck }')['healthCheck']
 print(json.dumps(report))
except Exception as error:
 report['capture']=(root/'traffic-capture.log').read_text()[-5000:] if (root/'traffic-capture.log').exists() else ''
 report['error']=str(error);print(json.dumps(report));raise
finally:
 db.execute('update routings set routing=? where selected=1',(original,));db.commit();db.close()
 db=sqlite3.connect(root/'state/wing.db')
 db.execute('update configs set global=? where selected=1',(original_global,))
 if test_group:
  db.execute('delete from group_policy_params where group_id=?',(test_group,))
  db.execute('delete from group_nodes where group_id=?',(test_group,));db.execute('delete from groups where id=?',(test_group,))
 if test_node:db.execute('delete from nodes where id=?',(test_node,))
 db.commit();db.close()
 if daemon_started:
  pid=int((root/'test.pid').read_text())
  def owned_process_alive():
   try:return pathlib.Path(f'/proc/{pid}/cmdline').read_bytes().startswith(str(root/'bin/dae-wing').encode())
   except FileNotFoundError:return False
  if owned_process_alive():
   subprocess.run(['kill','-TERM',str(pid)])
   for _ in range(50):
    if not owned_process_alive():break
    time.sleep(.1)
   report['gracefulShutdown']=not owned_process_alive()
   if owned_process_alive():subprocess.run(['kill','-KILL',str(pid)],stderr=subprocess.DEVNULL)
 for p in processes:
  if p.poll() is None:p.terminate();p.wait(timeout=5)
 for rule in firewall:run('iptables','-D',*rule)
 for ns in names:subprocess.run(['ip','netns','del',ns])
 if daemon_started:
  subprocess.run(['ip','link','del','dae0'],stderr=subprocess.DEVNULL)
  subprocess.run(['ip','netns','del','daens'],stderr=subprocess.DEVNULL)
 for key,value in saved.items():
  if '.daetest' not in key:run('sysctl','-qw',key+'='+value)
 report['cleanup']={'sysctlsRestored':all(run('sysctl','-n',k)==v for k,v in saved.items() if '.daetest' not in k),'githubStatus':run('curl','-s','-o','/dev/null','-w','%{http_code}','--max-time','15','https://github.com')}
 db=sqlite3.connect(root/'state/wing.db');db.execute('update systems set running=0');db.commit();db.close()
 (root/'traffic-report.json').write_text(json.dumps(report,indent=2))
 print('Cleanup',json.dumps(report['cleanup']))
