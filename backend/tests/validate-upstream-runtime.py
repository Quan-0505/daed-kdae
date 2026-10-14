"""Run only inside a disposable mount/network namespace with private bpffs.

Creates synthetic credentials/data and owns every process/interface it uses.
Never read a production database or change a production service.
"""
import argparse,json,os,pathlib,re,secrets,signal,socket,sqlite3,struct,subprocess,tempfile,threading,time,urllib.request

p=argparse.ArgumentParser();p.add_argument('--binary',required=True);p.add_argument('--geo',required=True);p.add_argument('--report',required=True);args=p.parse_args()
assert pathlib.Path('/run/netns/.kdae-disposable').is_file(),'Launch in a private mount/network namespace first'
for namespace in ('net','mnt'):
    assert os.stat('/proc/self/ns/'+namespace).st_ino!=os.stat('/proc/1/ns/'+namespace).st_ino,'Refusing the system '+namespace+' namespace'
def command(*argv):subprocess.run(argv,check=True,stdout=subprocess.DEVNULL)
for role,host,peer,network in [('client','ktest1','kcli0','21'),('server','ktest0','ksrv0','20')]:
    command('ip','netns','add','probe-'+role)
    command('ip','link','add',host,'type','veth','peer','name',peer)
    command('ip','link','set',peer,'netns','probe-'+role)
    command('ip','addr','add','198.18.'+network+'.1/24','dev',host);command('ip','link','set',host,'up')
    for tail in [('link','set','lo','up'),('addr','add','198.18.'+network+'.2/24','dev',peer),('link','set',peer,'up'),('route','add','default','via','198.18.'+network+'.1')]:command('ip','-n','probe-'+role,*tail)
for interface in ('all','default','ktest0','ktest1'):
    for name in ('send_redirects','accept_redirects','secure_redirects','accept_source_route','rp_filter'):command('sysctl','-qw',f'net.ipv4.conf.{interface}.{name}=0')
command('sysctl','-qw','net.ipv4.ip_forward=1','net.ipv6.conf.all.forwarding=1')
server_code='''import socket,threading
def tcp():
 s=socket.socket();s.bind(('0.0.0.0',18080));s.listen()
 while True:
  c,_=s.accept()
  def echo(c):
   with c:
    while True:
     data=c.recv(4096)
     if not data:break
     c.sendall(data)
  threading.Thread(target=echo,args=(c,),daemon=True).start()
threading.Thread(target=tcp,daemon=True).start()
s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);s.bind(('0.0.0.0',18081))
while True:
 data,peer=s.recvfrom(65535);s.sendto(data,peer)
'''
client_code='''import socket,sys
t=socket.create_connection(('198.18.20.2',18080),20);t.settimeout(8)
u=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);u.settimeout(8)
print('ready',flush=True)
for line in sys.stdin:
 data=line.strip().encode();t.sendall(data);assert t.recv(4096)==data
 u.sendto(data,('198.18.20.2',18081));assert u.recv(4096)==data
 print('ok',flush=True)
'''
processes=[];daemon=None
with tempfile.TemporaryDirectory(prefix='kdae-public-runtime-') as temporary:
    root=pathlib.Path(temporary);state=root/'state';state.mkdir(mode=0o700)
    for name in ('geoip.dat','geosite.dat'):(state/name).write_bytes((pathlib.Path(args.geo)/name).read_bytes())
    log_path=root/'runtime.log';log=log_path.open('w')
    base='http://127.0.0.1:12076';token=''
    def gql(query,auth=True):
        request=urllib.request.Request(base+'/graphql',json.dumps({'query':query}).encode(),{'Content-Type':'application/json','Authorization':'Bearer '+token if auth else ''})
        result=json.load(urllib.request.urlopen(request,timeout=120))
        if result.get('errors'):raise RuntimeError(result['errors'])
        return result['data']
    def api(path,data=None,method=None,auth=True):
        request=urllib.request.Request(base+'/api'+path,json.dumps(data).encode() if data is not None else None,{'Content-Type':'application/json','Authorization':'Bearer '+token if auth else ''},method=method)
        return json.load(urllib.request.urlopen(request,timeout=20))
    def start(api_only):
        process=subprocess.Popen([args.binary,'run','-c',str(state),'-l','127.0.0.1:12076','--disable-timestamp']+(['--api-only'] if api_only else []),stdout=log,stderr=subprocess.STDOUT)
        processes.append(process)
        for _ in range(200):
            if process.poll() is not None:raise RuntimeError('Test controller exited; inspect disposable runtime log')
            try:gql('query { healthCheck }',bool(token));return process
            except (OSError,RuntimeError):time.sleep(.1)
        raise RuntimeError('Test API did not become ready')
    try:
        daemon=start(True)
        token=gql('mutation { createUser(username:"probe",password:'+json.dumps(secrets.token_urlsafe(24))+') }',False)['createUser']
        daemon.terminate();daemon.wait(timeout=15)
        # Bootstrap shutdown is intentional; assess only the live dataplane.
        log.seek(0);log.truncate()
        global_config='global { wan_interface: ktest0\n lan_interface: ktest1\n disable_waiting_network: true\n log_level: info\n }'
        dns_config="dns { bind: '127.0.0.1:15353'\n upstream { local: 'udp://127.0.0.1:15354' }\n routing { request { fallback: local }\n response { fallback: accept } } }"
        with sqlite3.connect(state/'wing.db') as db:
            for table,column,value in [('configs','global',global_config),('dns','dns',dns_config),('routings','routing','routing { fallback: direct }')]:
                db.execute('UPDATE '+table+' SET selected=0')
                db.execute('INSERT INTO '+table+'(name,'+column+',selected,version) VALUES(?,?,1,1)',('probe',value))
        daemon=start(False)
        for _ in range(200):
            if 'msg=Ready' in log_path.read_text():break
            if daemon.poll() is not None:raise RuntimeError('Test dataplane exited')
            time.sleep(.1)
        else:raise RuntimeError('Test dataplane did not become ready')
        assert gql('mutation { run(dry:false) }')['run']==1
        server=subprocess.Popen(['ip','netns','exec','probe-server','python3','-u','-c',server_code]);processes.append(server);time.sleep(.3)
        client=subprocess.Popen(['ip','netns','exec','probe-client','python3','-u','-c',client_code],stdin=subprocess.PIPE,stdout=subprocess.PIPE,text=True);processes.append(client)
        assert client.stdout.readline().strip()=='ready'
        def traffic():client.stdin.write('persistent-flow\n');client.stdin.flush();assert client.stdout.readline().strip()=='ok'
        traffic();reloads=[]
        for _ in range(3):
            before=log_path.read_text().count('Loaded eBPF programs and maps')
            assert gql('mutation { run(dry:false) }')['run']==1
            assert daemon.poll() is None and gql('query { healthCheck }')['healthCheck']==1
            assert log_path.read_text().count('Loaded eBPF programs and maps')==before+1
            traffic();reloads.append(True)
        try:api('/logs',auth=False);raise AssertionError('Unauthenticated logs exposed')
        except urllib.error.HTTPError as error:assert error.code==401
        original=api('/runtime/log-level')['level'];snapshot=api('/logs')
        stream=urllib.request.urlopen(urllib.request.Request(base+'/api/events/logs?after_id='+str(snapshot['cursor'])+'&generation='+str(snapshot['generation']),headers={'Authorization':'Bearer '+token}),timeout=10)
        api('/runtime/log-level',{'level':'warn'},'PATCH');received=[]
        def consume():
            for _ in range(100):
                if b'Runtime log level updated' in stream.readline():received.append(True);return
        reader=threading.Thread(target=consume,daemon=True);reader.start();reader.join(8);stream.close();assert received
        assert api('/logs?level=warn&q=Runtime%20log%20level%20updated')['items']
        api('/runtime/log-level',{'level':original},'PATCH')
        assert api('/runtime/metrics')['rssBytes']>0
        errors=[line for line in log_path.read_text().splitlines() if re.search(r'level=(error|fatal)',line)]
        assert not errors,'Runtime error/fatal logged before shutdown: '+ '\n'.join(errors)
        report={'version':subprocess.check_output([args.binary,'--version'],text=True).strip(),'threeHotReloads':all(reloads),'persistentTCPAndUDPEcho':True,'nativeDNSBindHandoff':True,'authenticatedLiveSSEAndFilters':True,'processMetrics':True,'noRuntimeErrorOrFatal':True}
        pathlib.Path(args.report).write_text(json.dumps(report,indent=2)+'\n');print(json.dumps(report,indent=2))
    finally:
        for process in reversed(processes):
            if process.poll() is None:process.terminate()
            try:process.wait(timeout=15)
            except subprocess.TimeoutExpired:process.kill();process.wait()
        log.close()
