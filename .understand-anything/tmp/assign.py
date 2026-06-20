import json
inp = json.load(open('/home/vnkjd/Projects/bakery/.understand-anything/tmp/ua-arch-input.json'))
fn = inp['fileNodes']
def pth(n): return n.get('filePath') or n['id'].split(':',1)[1]

layers = {
 'layer:composition-root': [],
 'layer:inbound-adapters': [],
 'layer:application': [],
 'layer:domain': [],
 'layer:outbound-infra': [],
 'layer:shared-kernel': [],
 'layer:data': [],
 'layer:frontend': [],
 'layer:infrastructure': [],
 'layer:documentation': [],
 'layer:config': [],
}

def classify(n):
    p = pth(n); t = n['type']; parts = p.split('/')
    low = p.lower()
    # docker-compose service nodes
    if t == 'service':
        return 'layer:infrastructure'
    if t == 'pipeline':  # Makefile
        return 'layer:infrastructure'
    if t == 'table':
        return 'layer:data'
    if t == 'document':
        return 'layer:documentation'
    # config nodes
    if t == 'config':
        # frontend config goes with frontend
        if parts[0] == 'frontend':
            return 'layer:frontend'
        return 'layer:config'
    # files (.go, .js, .sql, etc)
    top = parts[0]
    if top == 'cmd' or (top=='internal' and len(parts)>1 and parts[1]=='deps'):
        return 'layer:composition-root'
    if top=='internal' and len(parts)>1 and parts[1]=='config':
        return 'layer:composition-root'
    if top=='internal' and len(parts)>1 and parts[1]=='inbound':
        return 'layer:inbound-adapters'
    if top=='internal' and len(parts)>1 and parts[1]=='outbound':
        return 'layer:outbound-infra'
    if top=='internal' and len(parts)>1 and parts[1]=='pkg':
        return 'layer:shared-kernel'
    if top=='pkg':
        return 'layer:shared-kernel'
    if top=='internal' and len(parts)>1 and parts[1]=='services':
        slice_ = parts[3] if len(parts)>3 else (parts[2] if len(parts)>2 else '')
        if slice_=='domain': return 'layer:domain'
        if slice_ in ('usecase','app'): return 'layer:application'
        if slice_=='infra': return 'layer:outbound-infra'
        return 'layer:application'
    if top=='migrations' or top=='queries' or low.endswith('.sql'):
        return 'layer:data'
    if top=='templates':
        return 'layer:data'
    if top=='frontend':
        return 'layer:frontend'
    if top=='deploy' or 'dockerfile' in low:
        return 'layer:infrastructure'
    # root-level leftover .go etc
    return 'layer:config'

unassigned=[]
for n in fn:
    L = classify(n)
    if L: layers[L].append(n['id'])
    else: unassigned.append(n['id'])

print('unassigned:', unassigned)
total=0
for L,ids in layers.items():
    print(f'{L}: {len(ids)}')
    total+=len(ids)
print('TOTAL', total, 'expected', len(fn))
json.dump(layers, open('/home/vnkjd/Projects/bakery/.understand-anything/tmp/layer-assign.json','w'))
