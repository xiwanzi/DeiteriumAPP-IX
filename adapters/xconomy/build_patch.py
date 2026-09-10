"""Build a deterministic XConomy Bukkit patch from the audited original jar.

Run Maven package first. The original binary is not committed to this repository.
GPL-3.0-or-later: all modified sources and the upstream licence accompany this tool.
"""
import argparse, hashlib, json, zipfile
from pathlib import Path

EXPECTED='0e3695f75f9d8769bb365d6c60fe48d169baf162b0bed423b456acef0a53584f'
def main():
    p=argparse.ArgumentParser();p.add_argument('--input',type=Path,required=True);p.add_argument('--output',type=Path,required=True);a=p.parse_args()
    if hashlib.sha256(a.input.read_bytes()).hexdigest()!=EXPECTED:raise SystemExit('Refusing unaudited XConomy input SHA-256')
    if a.input.resolve()==a.output.resolve():raise SystemExit('Input and output must differ')
    base=Path(__file__).resolve().parent;classes=base/'target/classes'
    modified={f.relative_to(classes).as_posix():f.read_bytes() for f in classes.rglob('*.class')}
    required=['data/DataCon.class','depend/economy/Vault.class','command/core/CommandPay.class','data/sql/SQL.class','deuterium/ControlledEconomyAPI.class']
    if any('me/yic/xconomy/'+name not in modified for name in required):raise SystemExit('Maven classes incomplete')
    with zipfile.ZipFile(a.input) as src:entries={n:src.read(n) for n in src.namelist() if not n.endswith('/') and not (n.startswith('META-INF/') and n.endswith(('.SF','.RSA','.DSA')))}
    plugin=entries['plugin.yml'].decode('utf-8')
    import re
    plugin,count=re.subn(r'(?m)^version:.*$',"version: '2.26.3-deuterium.3'",plugin)
    if count!=1:raise SystemExit('Unexpected plugin metadata')
    entries['plugin.yml']=plugin.encode();entries.update(modified)
    entries['META-INF/deuterium-xconomy-patch.json']=json.dumps({'version':'2.26.3-deuterium.3','inputSha256':EXPECTED,'apiVersion':1,'ledgerApiVersion':1,'mailCreditApiVersion':1,'modifiedClasses':sorted(modified)},sort_keys=True,indent=2).encode()
    entries['META-INF/DEUTERIUM-LICENSE']= (base/'LICENSE').read_bytes()
    a.output.parent.mkdir(parents=True,exist_ok=True)
    with zipfile.ZipFile(a.output,'w',zipfile.ZIP_DEFLATED,compresslevel=9) as out:
        for name,data in sorted(entries.items()):
            info=zipfile.ZipInfo(name,(2026,9,8,0,0,0));info.compress_type=zipfile.ZIP_DEFLATED;info.external_attr=0o644<<16;out.writestr(info,data)
    print(json.dumps({'artifact':str(a.output.resolve()),'sha256':hashlib.sha256(a.output.read_bytes()).hexdigest(),'modifiedClasses':len(modified)}))
if __name__=='__main__':main()
