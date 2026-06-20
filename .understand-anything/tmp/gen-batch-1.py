import json, math

batches=json.load(open('/home/vnkjd/Projects/bakery/.understand-anything/intermediate/batches.json'))
batches=batches if isinstance(batches,list) else batches.get('batches',[])
b=[x for x in batches if x.get('batchIndex')==1][0]
imp=b.get('batchImportData',{})
res=json.load(open('/home/vnkjd/Projects/bakery/.understand-anything/tmp/ua-file-extract-results-1.json'))['results']
resByPath={r['path']:r for r in res}

nodes=[]
edges=[]

# ---- File-level summaries (Russian) ----
file_meta={
 'internal/deps/infra.go':("Контейнер инфраструктурных зависимостей с builder-методами (config, Postgres, репозитории, RabbitMQ, iiko-клиент) для сборки приложения.",["dependency-injection","builder","infrastructure","wiring"],"moderate"),
 'internal/outbound/db/sqlc/auth.sql.go':("Сгенерированный sqlc-код доступа к данным пользователей: CRUD, привязка Telegram ID, выборки по роли и департаменту.",["data-access","sqlc-generated","auth","database"],"complex"),
 'internal/outbound/db/sqlc/db.go':("Базовый слой sqlc: интерфейс DBTX и тип Queries для исполнения запросов в рамках соединения или транзакции.",["data-access","sqlc-generated","database"],"simple"),
 'internal/outbound/db/sqlc/departments.sql.go':("Сгенерированный sqlc-код для департаментов: создание, выборки по коду/ID, привязка пользователя к департаменту.",["data-access","sqlc-generated","database"],"moderate"),
 'internal/outbound/db/sqlc/iiko_snapshot.sql.go':("Сгенерированный sqlc-код снапшотов iiko: запуски синхронизации, upsert продуктов и техкарт (assembly/prepared).",["data-access","sqlc-generated","iiko","database"],"complex"),
 'internal/outbound/db/sqlc/models.go':("Сгенерированные sqlc-модели таблиц БД (пользователи, заказы, департаменты, техкарты iiko, outbox).",["data-model","sqlc-generated","database","type-definition"],"moderate"),
 'internal/outbound/db/sqlc/monitor.sql.go':("Сгенерированный sqlc-код для монитора: выборка активных техкарт и их позиций по продукту.",["data-access","sqlc-generated","monitor","database"],"complex"),
 'internal/outbound/db/sqlc/order_outbox.sql.go':("Сгенерированный sqlc-код таблицы outbox заказов: вставка, выборка неопубликованных, пометка опубликованными.",["data-access","sqlc-generated","outbox","database"],"moderate"),
 'internal/outbound/db/sqlc/orders.sql.go':("Сгенерированный sqlc-код заказов: CRUD, позиции, история, счётчик номеров, выборки и пометка избранного.",["data-access","sqlc-generated","orders","database"],"complex"),
 'internal/outbound/db/sqlc/products.sql.go':("Сгенерированный sqlc-код каталога блюд и продуктов iiko: upsert, выборки по коду/имени, удаление.",["data-access","sqlc-generated","products","database"],"complex"),
 'internal/outbound/db/sqlc/querier.go':("Интерфейс Querier, агрегирующий все сгенерированные sqlc-методы доступа к данным.",["data-access","sqlc-generated","interface","database"],"moderate"),
 'internal/outbound/iiko/api.go':("Построитель URL-эндпоинтов iiko API (авторизация, номенклатура, техкарты) на основе базового адреса.",["api-client","url-builder","iiko"],"simple"),
 'internal/outbound/iiko/client.go':("HTTP-клиент iiko ERP: авторизация, выход, получение номенклатуры и техкарт (assembly/prepared) с JSON-декодированием.",["api-client","http","iiko","integration"],"complex"),
 'internal/outbound/iiko/client_test.go':("Интеграционный тест клиента iiko, последовательно вызывающий все методы и сохраняющий ответы в JSON.",["test","integration","iiko"],"moderate"),
 'internal/outbound/iiko/consts.go':("Константы для интеграции с iiko API.",["constants","iiko","configuration"],"simple"),
 'internal/outbound/iiko/dto.go':("DTO-структуры запросов и ответов iiko API: номенклатура, продукты, техкарты, ингредиентные карты.",["dto","type-definition","iiko","serialization"],"complex"),
 'internal/pkg/correlation/correlation.go':("Утилита correlation ID: запись/чтение из контекста и генерация при отсутствии для сквозной трассировки.",["utility","correlation","context","tracing"],"simple"),
 'internal/services/auth/app/app.go':("Композиция приложения сервиса авторизации: сборка usecase и RBAC из репозиториев.",["wiring","dependency-injection","auth","app-composition"],"simple"),
 'internal/services/auth/infra/repo/auth_repo.go':("Репозиторий авторизации поверх sqlc: CRUD пользователей, роли, привязка Telegram, департаменты, маппинг в домен.",["repository","data-access","auth","infrastructure"],"complex"),
 'internal/services/department/app/app.go':("Композиция приложения сервиса департаментов: сборка usecase из репозитория.",["wiring","dependency-injection","app-composition"],"simple"),
 'internal/services/department/infra/repo/department_repo.go':("Репозиторий департаментов поверх sqlc: выборки по типу, коду и ID с маппингом в домен.",["repository","data-access","infrastructure"],"moderate"),
 'internal/services/monitor/app/app.go':("Композиция приложения сервиса монитора: сборка usecase из репозитория.",["wiring","dependency-injection","app-composition"],"simple"),
 'internal/services/monitor/infra/repo/monitor_repo.go':("Репозиторий монитора: разрешение ингредиентов и загрузка графа техкарт заказа из снапшота iiko.",["repository","data-access","monitor","infrastructure"],"complex"),
 'internal/services/order/app/app.go':("Композиция приложения сервиса заказов: сборка usecase и outbox-relay из репозитория и публикатора.",["wiring","dependency-injection","orders","app-composition"],"simple"),
 'internal/services/order/infra/outbox/relay.go':("Outbox-relay: периодически читает неопубликованные события заказов и публикует их через Publisher.",["outbox","relay","messaging","infrastructure"],"moderate"),
 'internal/services/order/infra/repo/order_repo.go':("Репозиторий заказов поверх sqlc: транзакционное создание/обновление с outbox, позиции, история, каталог, избранное.",["repository","data-access","orders","outbox"],"complex"),
 'internal/services/sync/app/app.go':("Композиция приложения сервиса синхронизации: сборка usecase из iiko-клиента и репозитория.",["wiring","dependency-injection","sync","app-composition"],"simple"),
 'internal/services/sync/infra/repo/sync_repo.go':("Репозиторий синхронизации: сохранение снапшота iiko (продукты, техкарты) в рамках запуска синхронизации.",["repository","data-access","sync","infrastructure"],"complex"),
 'internal/services/sync/usecase/sync/interfaces.go':("Порты сервиса синхронизации: UseCase, IikoClient и Repository.",["interface","ports","sync","type-definition"],"simple"),
 'internal/services/sync/usecase/sync/sync.go':("Usecase синхронизации: периодический и однократный прогон выгрузки данных из iiko в снапшот.",["usecase","service","sync","scheduler"],"moderate"),
 'internal/services/techcard/app/app.go':("Композиция приложения сервиса техкарт: сборка usecase из репозитория.",["wiring","dependency-injection","app-composition"],"simple"),
 'internal/services/techcard/domain/model.go':("Доменные модели техкарт: TechCard и TechCardProduct.",["data-model","domain","type-definition"],"simple"),
 'internal/services/techcard/infra/repo/techcard_repo.go':("Репозиторий техкарт: сборка полной техкарты по коду из assembly/prepared карт снапшота iiko.",["repository","data-access","infrastructure"],"moderate"),
 'internal/services/techcard/usecase/techcard/interfaces.go':("Порты сервиса техкарт: UseCase и Repository.",["interface","ports","type-definition"],"simple"),
 'internal/services/techcard/usecase/techcard/techcard.go':("Usecase техкарт: получение техкарты по коду через репозиторий.",["usecase","service"],"simple"),
}

def basename(p): return p.rsplit('/',1)[-1]

for f in b['files']:
    p=f['path']
    summ,tags,cx=file_meta[p]
    nodes.append({"id":f"file:{p}","type":"file","name":basename(p),"filePath":p,"summary":summ,"tags":tags,"complexity":cx})

# ---- significant functions/classes ----
# significance: fn >=10 lines OR exported; class >=2 methods or >=20 lines OR exported
def fn_meta(path,name):
    # generic concise russian summary
    return (f"Функция {name}.", ["function"])

# Curated tags per file type for sub-nodes
sqlc_files={'internal/outbound/db/sqlc/auth.sql.go','internal/outbound/db/sqlc/departments.sql.go','internal/outbound/db/sqlc/iiko_snapshot.sql.go','internal/outbound/db/sqlc/monitor.sql.go','internal/outbound/db/sqlc/order_outbox.sql.go','internal/outbound/db/sqlc/orders.sql.go','internal/outbound/db/sqlc/products.sql.go'}

# We only emit function nodes for non-sqlc-generated meaningful code to keep graph clean,
# plus key exported classes. sqlc query functions are generated boilerplate -> skip per "auto-generated boilerplate" rule.
# But emit important structural classes (repositories, clients, services) and key functions.

func_nodes_spec=[]  # (path, name, start, end, summary, tags, cx)

def add_fn(path,name,summ,tags,cx="simple"):
    r=resByPath[path]
    for fn in r['functions']:
        if fn['name']==name:
            func_nodes_spec.append((path,name,fn['startLine'],fn['endLine'],summ,tags,cx))
            return
def add_cls(path,name,summ,tags,cx="moderate"):
    r=resByPath[path]
    for c in r['classes']:
        if c['name']==name:
            func_nodes_spec.append((path,name,c['startLine'],c['endLine'],summ,tags,cx,'class'))
            return

# infra.go
add_cls('internal/deps/infra.go','InfraDeps',"Контейнер инфраструктурных зависимостей приложения.",["dependency-injection","infrastructure"])
add_fn('internal/deps/infra.go','NewInfraDeps',"Создаёт контейнер инфраструктурных зависимостей.",["builder","dependency-injection"])
add_fn('internal/deps/infra.go','WithPostgres',"Инициализирует пул подключений к Postgres.",["builder","database"])
add_fn('internal/deps/infra.go','WithRabbitMQ',"Инициализирует подключение к RabbitMQ.",["builder","messaging"])
add_fn('internal/deps/infra.go','WithIikoClient',"Инициализирует клиент iiko API.",["builder","iiko"])

# iiko client
add_cls('internal/outbound/iiko/client.go','IikoClient',"Интерфейс HTTP-клиента iiko API.",["interface","api-client","iiko"])
add_cls('internal/outbound/iiko/client.go','Client',"HTTP-клиент iiko ERP.",["api-client","http","iiko"])
add_fn('internal/outbound/iiko/client.go','NewClient',"Создаёт клиент iiko с базовым URL и HTTP-клиентом.",["factory","iiko"])
add_fn('internal/outbound/iiko/client.go','Auth',"Авторизуется в iiko и сохраняет токен доступа.",["api-client","auth","iiko"])
add_fn('internal/outbound/iiko/client.go','Logout',"Завершает сессию в iiko.",["api-client","iiko"])
add_fn('internal/outbound/iiko/client.go','get',"Выполняет GET-запрос к iiko с обработкой ответа.",["http","api-client"])
add_fn('internal/outbound/iiko/client.go','ListProductsWithCategories',"Получает номенклатуру iiko с категориями продуктов.",["api-client","iiko"],"moderate")
add_fn('internal/outbound/iiko/client.go','AssemblyChartsGetAll',"Получает все техкарты сборки из iiko.",["api-client","iiko"])
add_fn('internal/outbound/iiko/client.go','AssemblyChartByID',"Получает техкарту сборки по ID из iiko.",["api-client","iiko"])
add_fn('internal/outbound/iiko/client.go','AssemblyChartsGetAssembled',"Получает собранные техкарты из iiko.",["api-client","iiko"])
add_fn('internal/outbound/iiko/client.go','AssemblyChartsGetPrepared',"Получает подготовленные техкарты из iiko.",["api-client","iiko"])

# iiko api
add_cls('internal/outbound/iiko/api.go','Api',"Построитель URL iiko API.",["url-builder","iiko"],"simple")
add_fn('internal/outbound/iiko/api.go','NewApi',"Создаёт построитель URL iiko на основе базового адреса.",["factory","iiko"])

# client_test
add_fn('internal/outbound/iiko/client_test.go','TestClient_AllMethods',"Интеграционный тест: проверяет все методы клиента iiko и сохраняет ответы.",["test","integration","iiko"],"moderate")

# correlation
add_fn('internal/pkg/correlation/correlation.go','EnsureID',"Возвращает correlation ID из контекста или генерирует новый.",["utility","correlation","context"])

# auth app
add_fn('internal/services/auth/app/app.go','New',"Собирает usecase авторизации из репозиториев.",["wiring","auth"])
add_fn('internal/services/auth/app/app.go','NewRBAC',"Собирает RBAC-компонент авторизации.",["wiring","auth","security"])

# auth repo
add_cls('internal/services/auth/infra/repo/auth_repo.go','AuthRepository',"Репозиторий авторизации поверх sqlc.",["repository","auth","data-access"])
add_fn('internal/services/auth/infra/repo/auth_repo.go','CreatePasswordUser',"Создаёт пользователя с паролем.",["repository","auth"])
add_fn('internal/services/auth/infra/repo/auth_repo.go','GetByUsername',"Возвращает пользователя по логину.",["repository","auth"])
add_fn('internal/services/auth/infra/repo/auth_repo.go','SetRole',"Устанавливает роль пользователя.",["repository","auth","rbac"])
add_fn('internal/services/auth/infra/repo/auth_repo.go','SetUsername',"Обновляет логин пользователя.",["repository","auth"])
add_fn('internal/services/auth/infra/repo/auth_repo.go','ListByRole',"Возвращает пользователей по роли.",["repository","auth","rbac"])
add_fn('internal/services/auth/infra/repo/auth_repo.go','ListByDepartmentID',"Возвращает пользователей департамента.",["repository","auth"])
add_fn('internal/services/auth/infra/repo/auth_repo.go','BindTelegramID',"Привязывает Telegram ID к пользователю.",["repository","auth","telegram"])
add_fn('internal/services/auth/infra/repo/auth_repo.go','SetPasswordHash',"Обновляет хеш пароля пользователя.",["repository","auth","security"])
add_fn('internal/services/auth/infra/repo/auth_repo.go','AssignUserDepartment',"Привязывает пользователя к департаменту.",["repository","auth"])
add_fn('internal/services/auth/infra/repo/auth_repo.go','authUserToDomain',"Маппит sqlc-модель пользователя в доменную сущность.",["mapper","auth"])

# department repo
add_cls('internal/services/department/infra/repo/department_repo.go','DepartmentRepository',"Репозиторий департаментов поверх sqlc.",["repository","data-access"])
add_fn('internal/services/department/infra/repo/department_repo.go','ListByType',"Возвращает департаменты по типу.",["repository"])

# monitor repo
add_cls('internal/services/monitor/infra/repo/monitor_repo.go','MonitorRepository',"Репозиторий монитора поверх снапшота iiko.",["repository","monitor","data-access"])
add_fn('internal/services/monitor/infra/repo/monitor_repo.go','ResolveIngredient',"Разрешает ингредиент продукта по активным техкартам.",["repository","monitor"])
add_fn('internal/services/monitor/infra/repo/monitor_repo.go','LoadOrderGraph',"Загружает граф техкарт заказа.",["repository","monitor"],"moderate")
add_fn('internal/services/monitor/infra/repo/monitor_repo.go','loadGraph',"Рекурсивно строит граф ингредиентов и техкарт.",["repository","monitor","recursion"],"complex")

# order app
add_fn('internal/services/order/app/app.go','New',"Собирает usecase заказов из репозитория.",["wiring","orders"])
add_fn('internal/services/order/app/app.go','NewOutboxRelay',"Собирает outbox-relay заказов.",["wiring","orders","outbox"])

# relay
add_cls('internal/services/order/infra/outbox/relay.go','Relay',"Outbox-relay публикации событий заказов.",["outbox","relay","messaging"])
add_cls('internal/services/order/infra/outbox/relay.go','Publisher',"Интерфейс публикатора событий.",["interface","messaging"],"simple")
add_fn('internal/services/order/infra/outbox/relay.go','Run',"Запускает периодический цикл публикации outbox-событий.",["outbox","relay","scheduler"])
add_fn('internal/services/order/infra/outbox/relay.go','drain',"Публикует пачку неопубликованных событий и помечает их.",["outbox","messaging"])

# order repo
add_cls('internal/services/order/infra/repo/order_repo.go','OrderRepository',"Репозиторий заказов поверх sqlc с транзакционным outbox.",["repository","orders","outbox","data-access"])
add_fn('internal/services/order/infra/repo/order_repo.go','CreateOrder',"Создаёт заказ в транзакции с записью в outbox.",["repository","orders","transaction"],"moderate")
add_fn('internal/services/order/infra/repo/order_repo.go','createOrderWithQueries',"Создаёт заказ и связанные сущности в рамках транзакции.",["repository","orders","transaction"],"moderate")
add_fn('internal/services/order/infra/repo/order_repo.go','persistOutbox',"Сохраняет доменные события заказа в outbox.",["repository","orders","outbox"])
add_fn('internal/services/order/infra/repo/order_repo.go','UpdateOrder',"Обновляет заказ в транзакции.",["repository","orders","transaction"])
add_fn('internal/services/order/infra/repo/order_repo.go','updateOrderTx',"Транзакционное обновление заказа.",["repository","orders","transaction"])
add_fn('internal/services/order/infra/repo/order_repo.go','updateOrderWithQueries',"Обновляет заказ и связанные сущности в транзакции.",["repository","orders","transaction"],"moderate")
add_fn('internal/services/order/infra/repo/order_repo.go','ListOrders',"Возвращает список заказов с позициями (без N+1).",["repository","orders"],"moderate")
add_fn('internal/services/order/infra/repo/order_repo.go','ResolveDishCatalogItem',"Разрешает позицию каталога блюд.",["repository","orders","catalog"])
add_fn('internal/services/order/infra/repo/order_repo.go','SetOrderFavorite',"Помечает заказ избранным.",["repository","orders"])
add_fn('internal/services/order/infra/repo/order_repo.go','createOrderItems',"Создаёт позиции заказа.",["repository","orders"])
add_fn('internal/services/order/infra/repo/order_repo.go','createOrderHistory',"Создаёт запись истории заказа.",["repository","orders"])
add_fn('internal/services/order/infra/repo/order_repo.go','listOrderHistory',"Загружает историю заказа с позициями.",["repository","orders"],"moderate")
add_fn('internal/services/order/infra/repo/order_repo.go','orderItemsByOrderIDs',"Загружает позиции для набора заказов одним запросом.",["repository","orders","batch"])

# sync app
add_fn('internal/services/sync/app/app.go','New',"Собирает usecase синхронизации из iiko-клиента и репозитория.",["wiring","sync"])

# sync repo
add_cls('internal/services/sync/infra/repo/sync_repo.go','SyncRepository',"Репозиторий синхронизации снапшота iiko.",["repository","sync","data-access"])
add_fn('internal/services/sync/infra/repo/sync_repo.go','SaveSnapshot',"Сохраняет снапшот данных iiko в рамках запуска синхронизации.",["repository","sync"],"moderate")
add_fn('internal/services/sync/infra/repo/sync_repo.go','saveSnapshotData',"Сохраняет продукты и техкарты снапшота iiko.",["repository","sync"],"complex")

# sync interfaces
add_cls('internal/services/sync/usecase/sync/interfaces.go','IikoClient',"Порт клиента iiko для синхронизации.",["interface","ports","iiko"],"simple")
add_cls('internal/services/sync/usecase/sync/interfaces.go','Repository',"Порт репозитория синхронизации.",["interface","ports"],"simple")
add_cls('internal/services/sync/usecase/sync/interfaces.go','UseCase',"Порт usecase синхронизации.",["interface","ports"],"simple")

# sync usecase
add_cls('internal/services/sync/usecase/sync/sync.go','Service',"Сервис синхронизации данных iiko.",["service","sync"])
add_fn('internal/services/sync/usecase/sync/sync.go','Run',"Запускает периодический цикл синхронизации.",["service","sync","scheduler"])
add_fn('internal/services/sync/usecase/sync/sync.go','SyncOnce',"Выполняет однократную синхронизацию данных из iiko.",["service","sync"],"moderate")

# techcard domain
add_cls('internal/services/techcard/domain/model.go','TechCard',"Доменная модель техкарты.",["data-model","domain"],"simple")
add_cls('internal/services/techcard/domain/model.go','TechCardProduct',"Доменная модель продукта техкарты.",["data-model","domain"],"simple")

# techcard repo
add_cls('internal/services/techcard/infra/repo/techcard_repo.go','TechCardRepository',"Репозиторий техкарт поверх снапшота iiko.",["repository","data-access"])
add_fn('internal/services/techcard/infra/repo/techcard_repo.go','GetByCode',"Собирает полную техкарту по коду.",["repository"],"moderate")
add_fn('internal/services/techcard/infra/repo/techcard_repo.go','attachAssembly',"Прикрепляет позиции техкарты сборки.",["repository"])
add_fn('internal/services/techcard/infra/repo/techcard_repo.go','attachPrepared',"Прикрепляет позиции подготовленной техкарты.",["repository"])
add_fn('internal/services/techcard/infra/repo/techcard_repo.go','addProduct',"Добавляет продукт в техкарту.",["repository"])

# techcard interfaces
add_cls('internal/services/techcard/usecase/techcard/interfaces.go','UseCase',"Порт usecase техкарт.",["interface","ports"],"simple")
add_cls('internal/services/techcard/usecase/techcard/interfaces.go','Repository',"Порт репозитория техкарт.",["interface","ports"],"simple")

# techcard usecase
add_cls('internal/services/techcard/usecase/techcard/techcard.go','Service',"Сервис техкарт.",["service"],"simple")
add_fn('internal/services/techcard/usecase/techcard/techcard.go','GetByCode',"Возвращает техкарту по коду через репозиторий.",["service"])

# build sub-nodes + contains/exports edges
def exp_names(r):
    out=set()
    for e in r.get('exports',[]):
        out.add(e['name'] if isinstance(e,dict) else e)
    return out
exported_by_file={r['path']:exp_names(r) for r in res}
for spec in func_nodes_spec:
    path,name,start,end,summ,tags=spec[0],spec[1],spec[2],spec[3],spec[4],spec[5]
    cx='simple'
    kind='function'
    for x in spec[6:]:
        if x in ('simple','moderate','complex'): cx=x
        elif x=='class': kind='class'
    nid=f"{kind}:{path}:{name}"
    nodes.append({"id":nid,"type":kind,"name":name,"filePath":path,"lineRange":[start,end],"summary":summ,"tags":tags,"complexity":cx})
    edges.append({"source":f"file:{path}","target":nid,"type":"contains","direction":"forward","weight":1.0})
    if name in exported_by_file.get(path,set()):
        edges.append({"source":f"file:{path}","target":nid,"type":"exports","direction":"forward","weight":0.8})

# imports edges
for src,targets in imp.items():
    for t in targets:
        edges.append({"source":f"file:{src}","target":f"file:{t}","type":"imports","direction":"forward","weight":0.7})

# tested_by: client_test.go tests iiko client.go (production -> test)
edges.append({"source":"file:internal/outbound/iiko/client.go","target":"file:internal/outbound/iiko/client_test.go","type":"tested_by","direction":"forward","weight":0.5})

# self-check import count
import_count=sum(len(v) for v in imp.values())
emitted_imports=sum(1 for e in edges if e['type']=='imports')
assert import_count==emitted_imports, f"{import_count} vs {emitted_imports}"

print("nodes",len(nodes),"edges",len(edges),"imports",emitted_imports)

# ---- partition ----
def node_file(n):
    return n.get('filePath')

node_count=len(nodes); edge_count=len(edges)

# per-file node/edge load
files_sorted=sorted([f['path'] for f in b['files']])
nodes_by_file={p:[n for n in nodes if node_file(n)==p] for p in files_sorted}
node_ids_by_file={p:set(n['id'] for n in nodes_by_file[p]) for p in files_sorted}
edges_by_file={}
for e in edges:
    # attribute edge to the file of its source node
    src=e['source']
    owner=None
    for p in files_sorted:
        if src in node_ids_by_file[p] or src==f"file:{p}":
            owner=p; break
    edges_by_file.setdefault(owner,[]).append(e)

NCAP=60; ECAP=120
groups=[]
cur=set(); cn=0; ce=0
for p in files_sorted:  # keep alphabetical order, sequential bin assignment
    fn=len(nodes_by_file[p]); fe=len(edges_by_file.get(p,[]))
    if cur and (cn+fn>NCAP or ce+fe>ECAP):
        groups.append(cur); cur=set(); cn=0; ce=0
    cur.add(p); cn+=fn; ce+=fe
if cur: groups.append(cur)
parts=len(groups)

written=[]
for k,g in enumerate(groups,1):
    pn=[n for n in nodes if node_file(n) in g]
    pn_ids=set(n['id'] for n in pn)
    pe=[e for e in edges if e['source'] in pn_ids]
    out={"nodes":pn,"edges":pe}
    fname=f"/home/vnkjd/Projects/bakery/.understand-anything/intermediate/batch-1{'' if parts==1 else f'-part-{k}'}.json"
    json.dump(out,open(fname,'w'),ensure_ascii=False,indent=1)
    written.append((fname,len(pn),len(pe)))

print("PARTS",parts)
for w in written: print(w)
print("TOTAL nodes",len(nodes),"edges",len(edges))
