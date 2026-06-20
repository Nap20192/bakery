import fs from "fs";

const raw = JSON.parse(fs.readFileSync(".understand-anything/tmp/ua-batch-3-raw.json","utf8"));
const imp = raw.batchImportData;

// File nodes (hand-authored Russian summaries/tags/complexity)
const nodes = [
  // deps.go
  { id:"file:internal/deps/deps.go", type:"file", name:"deps.go", filePath:"internal/deps/deps.go",
    summary:"Контейнер зависимостей приложения с функциональными опциями (With...) для сборки всех сервисов, бота, outbox-relay и HTTP-сервера из инфраструктуры.",
    tags:["dependency-injection","factory","composition-root","wiring"], complexity:"moderate",
    languageNotes:"Паттерн functional options: каждая With*-функция возвращает опцию, мутирующую AppDeps." },
  { id:"class:internal/deps/deps.go:AppDeps", type:"class", name:"AppDeps", filePath:"internal/deps/deps.go", lineRange:[25,37],
    summary:"Структура-агрегатор со ссылками на все доменные сервисы, outbox-relay, API-сервер и Telegram-бот.", tags:["data-model","composition-root","container"], complexity:"simple" },
  { id:"function:internal/deps/deps.go:WithOrderBot", type:"function", name:"WithOrderBot", filePath:"internal/deps/deps.go", lineRange:[138,170],
    summary:"Опция сборки: валидирует обязательные сервисы и создаёт OrderBot со всеми бэкендами и mini-app URL.", tags:["factory","wiring","validation"], complexity:"moderate" },
  { id:"function:internal/deps/deps.go:WithAPIServerConfig", type:"function", name:"WithAPIServerConfig", filePath:"internal/deps/deps.go", lineRange:[172,187],
    summary:"Опция сборки: настраивает и создаёт HTTP API-сервер на основе конфигурации и доменных сервисов.", tags:["factory","wiring","api-handler"], complexity:"simple" },

  // apperror.go
  { id:"file:internal/inbound/api/httpx/apperror.go", type:"file", name:"apperror.go", filePath:"internal/inbound/api/httpx/apperror.go",
    summary:"Преобразует доменные apperr.Error в HTTP-ответы: маппит kind на статус-код и логирует внутренние ошибки.", tags:["error-handling","api-handler","serialization"], complexity:"simple" },
  { id:"function:internal/inbound/api/httpx/apperror.go:WriteAppError", type:"function", name:"WriteAppError", filePath:"internal/inbound/api/httpx/apperror.go", lineRange:[23,37],
    summary:"Записывает доменную ошибку в HTTP-ответ, выбирая статус по apperr.Kind и логируя серверные сбои.", tags:["error-handling","api-handler"], complexity:"simple" },

  // auth.go
  { id:"file:internal/inbound/api/httpx/auth.go", type:"file", name:"auth.go", filePath:"internal/inbound/api/httpx/auth.go",
    summary:"Аутентификация HTTP API: middleware для Telegram mini-app (валидация initData по HMAC) и Bearer-токенов, резолвинг пользователя и построение viewer с данными подразделения.",
    tags:["middleware","authentication","security","api-handler"], complexity:"complex",
    languageNotes:"Проверка mini-app initData реализует HMAC-SHA256 схему Telegram WebApp с проверкой срока действия auth_date." },
  { id:"class:internal/inbound/api/httpx/auth.go:Authenticator", type:"class", name:"Authenticator", filePath:"internal/inbound/api/httpx/auth.go", lineRange:[52,56],
    summary:"Содержит auth- и department-сервисы и bot-токен; предоставляет middleware RequireAuth/RequireMiniAppAuth/RequireAdmin.", tags:["middleware","authentication","security"], complexity:"complex" },
  { id:"class:internal/inbound/api/httpx/auth.go:MiniAppUser", type:"class", name:"MiniAppUser", filePath:"internal/inbound/api/httpx/auth.go", lineRange:[35,43],
    summary:"DTO аутентифицированного mini-app пользователя с данными Telegram и подразделения, помещаемый в контекст запроса.", tags:["data-model","authentication"], complexity:"simple" },
  { id:"function:internal/inbound/api/httpx/auth.go:RequireMiniAppAuth", type:"function", name:"RequireMiniAppAuth", filePath:"internal/inbound/api/httpx/auth.go", lineRange:[68,86],
    summary:"Middleware: требует валидной аутентификации mini-app, резолвит пользователя и кладёт его в контекст.", tags:["middleware","authentication","security"], complexity:"moderate" },
  { id:"function:internal/inbound/api/httpx/auth.go:RequireAdmin", type:"function", name:"RequireAdmin", filePath:"internal/inbound/api/httpx/auth.go", lineRange:[106,123],
    summary:"Middleware: пропускает только пользователей с ролью администратора.", tags:["middleware","authentication","security","rbac"], complexity:"moderate" },
  { id:"function:internal/inbound/api/httpx/auth.go:resolveUser", type:"function", name:"resolveUser", filePath:"internal/inbound/api/httpx/auth.go", lineRange:[130,157],
    summary:"Определяет пользователя по заголовку Authorization: либо по mini-app initData, либо по Bearer-токену.", tags:["authentication","security"], complexity:"moderate" },
  { id:"function:internal/inbound/api/httpx/auth.go:buildDepartmentViewer", type:"function", name:"buildDepartmentViewer", filePath:"internal/inbound/api/httpx/auth.go", lineRange:[171,200],
    summary:"Дополняет MiniAppUser данными подразделения, запрашивая department-сервис по ID.", tags:["authentication","data-model"], complexity:"moderate" },
  { id:"function:internal/inbound/api/httpx/auth.go:validateMiniAppInitData", type:"function", name:"validateMiniAppInitData", filePath:"internal/inbound/api/httpx/auth.go", lineRange:[242,294],
    summary:"Проверяет подпись Telegram mini-app initData по HMAC-SHA256 и срок действия auth_date, возвращая данные пользователя.", tags:["security","validation","authentication"], complexity:"complex",
    languageNotes:"Двухступенчатый HMAC: сначала секретный ключ из bot-токена, затем подпись отсортированных пар key=value." },

  // auth_test.go
  { id:"file:internal/inbound/api/httpx/auth_test.go", type:"file", name:"auth_test.go", filePath:"internal/inbound/api/httpx/auth_test.go",
    summary:"Тесты валидации mini-app initData (включая просроченные подписи) и разбора Authorization-заголовка.", tags:["test","authentication","security"], complexity:"moderate" },

  // respond.go
  { id:"file:internal/inbound/api/httpx/respond.go", type:"file", name:"respond.go", filePath:"internal/inbound/api/httpx/respond.go",
    summary:"HTTP-хелперы: запись JSON/ошибок, общие DTO ответов и разбор дат и уникальных query-параметров запроса.", tags:["utility","serialization","api-handler"], complexity:"moderate" },
  { id:"function:internal/inbound/api/httpx/respond.go:ParseRequestDate", type:"function", name:"ParseRequestDate", filePath:"internal/inbound/api/httpx/respond.go", lineRange:[51,63],
    summary:"Разбирает дату запроса в форматах RFC3339 и YYYY-MM-DD, возвращая ошибку при неверном формате.", tags:["utility","validation","date"], complexity:"simple" },
  { id:"function:internal/inbound/api/httpx/respond.go:UniqueQueryValues", type:"function", name:"UniqueQueryValues", filePath:"internal/inbound/api/httpx/respond.go", lineRange:[66,81],
    summary:"Возвращает дедуплицированный список непустых значений query-параметра с сохранением порядка.", tags:["utility"], complexity:"simple" },

  // middleware.go (api)
  { id:"file:internal/inbound/api/middleware.go", type:"file", name:"middleware.go", filePath:"internal/inbound/api/middleware.go",
    summary:"HTTP middleware сервера: CORS с белым списком origin, recover от паник и логирование запросов.", tags:["middleware","api-handler","cors","logging"], complexity:"moderate" },
  { id:"function:internal/inbound/api/middleware.go:cors", type:"function", name:"cors", filePath:"internal/inbound/api/middleware.go", lineRange:[16,32],
    summary:"CORS middleware: выставляет заголовки доступа для разрешённых origin и обрабатывает preflight-запросы.", tags:["middleware","cors","security"], complexity:"moderate" },
  { id:"function:internal/inbound/api/middleware.go:recoverer", type:"function", name:"recoverer", filePath:"internal/inbound/api/middleware.go", lineRange:[34,44],
    summary:"Middleware-перехватчик паник: логирует panic и возвращает 500 вместо падения сервера.", tags:["middleware","error-handling"], complexity:"simple" },

  // server.go
  { id:"file:internal/inbound/api/server.go", type:"file", name:"server.go", filePath:"internal/inbound/api/server.go",
    summary:"HTTP API-сервер: собирает обработчики всех сервисов, регистрирует маршруты с middleware и управляет жизненным циклом (Start/Shutdown).",
    tags:["entry-point","api-handler","server","wiring"], complexity:"moderate" },
  { id:"class:internal/inbound/api/server.go:Server", type:"class", name:"Server", filePath:"internal/inbound/api/server.go", lineRange:[33,38],
    summary:"HTTP-сервер с аутентификатором, набором регистраторов маршрутов и конфигурацией; поддерживает graceful shutdown.", tags:["server","api-handler","wiring"], complexity:"moderate" },
  { id:"function:internal/inbound/api/server.go:NewServer", type:"function", name:"NewServer", filePath:"internal/inbound/api/server.go", lineRange:[40,60],
    summary:"Создаёт Server: строит presenter, аутентификатор и HTTP-обработчики admin/auth/department/order/monitor сервисов.", tags:["factory","wiring","api-handler"], complexity:"moderate" },
  { id:"function:internal/inbound/api/server.go:Start", type:"function", name:"Start", filePath:"internal/inbound/api/server.go", lineRange:[62,77],
    summary:"Регистрирует health-check и маршруты сервисов, оборачивает в middleware и запускает прослушивание HTTP.", tags:["server","api-handler","entry-point"], complexity:"moderate" },

  // action_menu.go
  { id:"file:internal/inbound/bot/action_menu.go", type:"file", name:"action_menu.go", filePath:"internal/inbound/bot/action_menu.go",
    summary:"Построение reply-клавиатуры Telegram-бота в зависимости от роли пользователя, типа подразделения и состояния сессии заказа.",
    tags:["bot","ui","keyboard","rbac"], complexity:"complex",
    languageNotes:"actionMenuSnapshot захватывает неизменяемый снимок состояния сессии, чтобы строить раскладку вне блокировки мьютекса." },
  { id:"class:internal/inbound/bot/action_menu.go:actionMenuSnapshot", type:"class", name:"actionMenuSnapshot", filePath:"internal/inbound/bot/action_menu.go", lineRange:[48,56],
    summary:"Снимок состояния сессии (роль, тип подразделения, элементы заказа, фильтры) для построения раскладки клавиатуры.", tags:["data-model","bot","ui"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/action_menu.go:actionMenu", type:"function", name:"actionMenu", filePath:"internal/inbound/bot/action_menu.go", lineRange:[62,92],
    summary:"Формирует снимок состояния для текущего пользователя под блокировкой сессии для дальнейшего рендера меню.", tags:["bot","ui","session"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/action_menu.go:resolveState", type:"function", name:"resolveState", filePath:"internal/inbound/bot/action_menu.go", lineRange:[94,124],
    summary:"Определяет текстовое описание текущего состояния меню (заказ, фильтр) для отображения пользователю.", tags:["bot","ui"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/action_menu.go:rows", type:"function", name:"rows", filePath:"internal/inbound/bot/action_menu.go", lineRange:[130,150],
    summary:"Собирает строки кнопок меню в зависимости от типа подразделения, прав и наличия фильтра/заказа.", tags:["bot","ui","keyboard"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/action_menu.go:orderShopFilterReplyRows", type:"function", name:"orderShopFilterReplyRows", filePath:"internal/inbound/bot/action_menu.go", lineRange:[185,209],
    summary:"Строит ряды кнопок-фильтров по магазинам, запрашивая список подразделений из department-сервиса.", tags:["bot","ui","keyboard"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/action_menu.go:actionKeyboard", type:"function", name:"actionKeyboard", filePath:"internal/inbound/bot/action_menu.go", lineRange:[211,235],
    summary:"Собирает финальную reply-клавиатуру из строк кнопок и кнопки запуска mini-app.", tags:["bot","ui","keyboard"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/action_menu.go:currentUser", type:"function", name:"currentUser", filePath:"internal/inbound/bot/action_menu.go", lineRange:[237,252],
    summary:"Возвращает аутентифицированного пользователя из кэша контекста или загружает его из auth-сервиса по Telegram ID.", tags:["bot","authentication","session"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/action_menu.go:userDepartmentType", type:"function", name:"userDepartmentType", filePath:"internal/inbound/bot/action_menu.go", lineRange:[254,263],
    summary:"Определяет тип подразделения пользователя по его department ID или роли как запасной вариант.", tags:["bot","rbac"], complexity:"simple" },

  // action_menu_test.go
  { id:"file:internal/inbound/bot/action_menu_test.go", type:"file", name:"action_menu_test.go", filePath:"internal/inbound/bot/action_menu_test.go",
    summary:"Тесты построения строк меню, резолвинга состояния и сопоставления роли с типом подразделения.", tags:["test","bot","ui"], complexity:"moderate" },

  // bot.go
  { id:"file:internal/inbound/bot/bot.go", type:"file", name:"bot.go", filePath:"internal/inbound/bot/bot.go",
    summary:"Конструктор и жизненный цикл Telegram-бота заказов: инициализация telebot, хранилище сессий и регистрация бэкендов сервисов.",
    tags:["bot","entry-point","wiring"], complexity:"moderate" },
  { id:"class:internal/inbound/bot/bot.go:OrderBot", type:"class", name:"OrderBot", filePath:"internal/inbound/bot/bot.go", lineRange:[29,33],
    summary:"Telegram-бот заказов с потокобезопасным хранилищем пользовательских сессий и ссылками на доменные сервисы.", tags:["bot","data-model","wiring"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/bot.go:NewOrderBot", type:"function", name:"NewOrderBot", filePath:"internal/inbound/bot/bot.go", lineRange:[35,79],
    summary:"Создаёт OrderBot: валидирует mini-app URL, инициализирует telebot-клиента и регистрирует маршруты.", tags:["bot","factory","wiring"], complexity:"moderate" },

  // handler_start.go
  { id:"file:internal/inbound/bot/handler_start.go", type:"file", name:"handler_start.go", filePath:"internal/inbound/bot/handler_start.go",
    summary:"Обработчики команд бота /start и /help: приветствие, сброс сессии и попытка автоаутентификации по Telegram.", tags:["bot","event-handler","authentication"], complexity:"simple" },
  { id:"function:internal/inbound/bot/handler_start.go:handleStart", type:"function", name:"handleStart", filePath:"internal/inbound/bot/handler_start.go", lineRange:[9,29],
    summary:"Обрабатывает /start: сбрасывает сессию, аутентифицирует пользователя по Telegram ID и показывает меню действий.", tags:["bot","event-handler","authentication"], complexity:"moderate" },

  // handler_templates.go
  { id:"file:internal/inbound/bot/handler_templates.go", type:"file", name:"handler_templates.go", filePath:"internal/inbound/bot/handler_templates.go",
    summary:"Обработчики бота для шаблонов заказов: список шаблонов, объединённый шаблон и применение выбранного шаблона.", tags:["bot","event-handler","order"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/handler_templates.go:handleTemplates", type:"function", name:"handleTemplates", filePath:"internal/inbound/bot/handler_templates.go", lineRange:[12,35],
    summary:"Показывает список доступных шаблонов заказов в виде inline-клавиатуры, запрашивая их из order-сервиса.", tags:["bot","event-handler","order","ui"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/handler_templates.go:handleTemplateAll", type:"function", name:"handleTemplateAll", filePath:"internal/inbound/bot/handler_templates.go", lineRange:[37,53],
    summary:"Обрабатывает выбор объединённого шаблона: получает сводный шаблон из order-сервиса и отображает его.", tags:["bot","event-handler","order"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/handler_templates.go:handleTemplateUse", type:"function", name:"handleTemplateUse", filePath:"internal/inbound/bot/handler_templates.go", lineRange:[55,72],
    summary:"Применяет выбранный шаблон заказа по коду из callback, загружая его из order-сервиса.", tags:["bot","event-handler","order"], complexity:"moderate" },

  // messages.go
  { id:"file:internal/inbound/bot/messages.go", type:"file", name:"messages.go", filePath:"internal/inbound/bot/messages.go",
    summary:"Константы текстов сообщений бота (предельно малый вспомогательный файл).", tags:["bot","constants","utility"], complexity:"simple" },

  // middleware.go (bot)
  { id:"file:internal/inbound/bot/middleware.go", type:"file", name:"middleware.go", filePath:"internal/inbound/bot/middleware.go",
    summary:"Middleware Telegram-бота: проверка RBAC-прав, ограничение приватными чатами и загрузка аутентифицированного пользователя в контекст.",
    tags:["bot","middleware","rbac","authentication"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/middleware.go:requirePermissions", type:"function", name:"requirePermissions", filePath:"internal/inbound/bot/middleware.go", lineRange:[16,34],
    summary:"Middleware: проверяет наличие требуемых RBAC-прав у пользователя перед выполнением обработчика.", tags:["bot","middleware","rbac","security"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/middleware.go:privateChatOnly", type:"function", name:"privateChatOnly", filePath:"internal/inbound/bot/middleware.go", lineRange:[36,61],
    summary:"Middleware: блокирует команды в групповых чатах, разрешая только приватные диалоги.", tags:["bot","middleware","security"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/middleware.go:authUserFromContext", type:"function", name:"authUserFromContext", filePath:"internal/inbound/bot/middleware.go", lineRange:[63,87],
    summary:"Загружает аутентифицированного пользователя из auth-сервиса по Telegram ID и кэширует в контексте.", tags:["bot","authentication","session"], complexity:"moderate" },

  // mini_app.go
  { id:"file:internal/inbound/bot/mini_app.go", type:"file", name:"mini_app.go", filePath:"internal/inbound/bot/mini_app.go",
    summary:"Построение deep-link на Telegram mini-app с параметрами экрана и номеров заказов и создание кнопки WebApp.", tags:["bot","mini-app","ui","utility"], complexity:"simple" },
  { id:"function:internal/inbound/bot/mini_app.go:miniAppLink", type:"function", name:"miniAppLink", filePath:"internal/inbound/bot/mini_app.go", lineRange:[18,43],
    summary:"Строит URL mini-app, добавляя query-параметры режима и номеров заказов к настроенному базовому адресу.", tags:["bot","mini-app","utility"], complexity:"moderate" },

  // mini_app_test.go
  { id:"file:internal/inbound/bot/mini_app_test.go", type:"file", name:"mini_app_test.go", filePath:"internal/inbound/bot/mini_app_test.go",
    summary:"Тесты построения mini-app ссылки: включение параметров экрана/заказов и пустой результат без настроенного URL.", tags:["test","bot","mini-app"], complexity:"simple" },

  // sender.go
  { id:"file:internal/inbound/bot/sender.go", type:"file", name:"sender.go", filePath:"internal/inbound/bot/sender.go",
    summary:"Отправка сообщений Telegram с разбивкой длинных текстов на чанки, сохранением балансировки <pre>-тегов и прикреплением клавиатуры только к последнему чанку.",
    tags:["bot","messaging","utility"], complexity:"complex",
    languageNotes:"splitTelegramMessage учитывает лимит длины сообщения Telegram и не разрывает <pre>-блоки." },
  { id:"function:internal/inbound/bot/sender.go:sendTelegramChunks", type:"function", name:"sendTelegramChunks", filePath:"internal/inbound/bot/sender.go", lineRange:[80,91],
    summary:"Разбивает сообщение на чанки и отправляет их через переданную функцию, прикрепляя опции только к последнему.", tags:["bot","messaging"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/sender.go:splitTelegramMessage", type:"function", name:"splitTelegramMessage", filePath:"internal/inbound/bot/sender.go", lineRange:[111,143],
    summary:"Разбивает HTML-сообщение на части в пределах лимита Telegram, сохраняя целостность <pre>-блоков.", tags:["bot","messaging","utility"], complexity:"complex" },
  { id:"function:internal/inbound/bot/sender.go:splitPlainText", type:"function", name:"splitPlainText", filePath:"internal/inbound/bot/sender.go", lineRange:[159,183],
    summary:"Разбивает обычный текст на части по строкам в пределах заданного лимита длины.", tags:["bot","messaging","utility"], complexity:"moderate" },
  { id:"function:internal/inbound/bot/sender.go:botSender", type:"function", name:"botSender", filePath:"internal/inbound/bot/sender.go", lineRange:[54,71],
    summary:"Возвращает функцию отправки сообщений в конкретный чат через telebot-клиент с логированием.", tags:["bot","messaging"], complexity:"moderate" },

  // sender_test.go
  { id:"file:internal/inbound/bot/sender_test.go", type:"file", name:"sender_test.go", filePath:"internal/inbound/bot/sender_test.go",
    summary:"Тесты отправки чанков (типизированные опции, прикрепление клавиатуры к последнему чанку), балансировки <pre>-тегов и блокировки групповых чатов.", tags:["test","bot","messaging"], complexity:"moderate" },

  // apperr.go
  { id:"file:internal/pkg/apperr/apperr.go", type:"file", name:"apperr.go", filePath:"internal/pkg/apperr/apperr.go",
    summary:"Доменный тип ошибок приложения с классификацией (Kind), кодом и сообщением, плюс конструкторы Invalid/NotFound/Conflict/Unauthorized/Forbidden/Internal.",
    tags:["error-handling","utility","type-definition"], complexity:"moderate",
    languageNotes:"Реализует интерфейсы error и Unwrap для совместимости с errors.As/Is." },
  { id:"class:internal/pkg/apperr/apperr.go:Error", type:"class", name:"Error", filePath:"internal/pkg/apperr/apperr.go", lineRange:[38,43],
    summary:"Структура ошибки с полями Kind, Code, Message и обёрнутой причиной; реализует error и Unwrap.", tags:["error-handling","data-model","type-definition"], complexity:"simple" },

  // apperr_test.go
  { id:"file:internal/pkg/apperr/apperr_test.go", type:"file", name:"apperr_test.go", filePath:"internal/pkg/apperr/apperr_test.go",
    summary:"Тесты доменного пакета ошибок: проверка классификации Kind, извлечения сообщения и обёртывания причин.", tags:["test","error-handling"], complexity:"simple" },

  // authtoken.go
  { id:"file:internal/pkg/authtoken/authtoken.go", type:"file", name:"authtoken.go", filePath:"internal/pkg/authtoken/authtoken.go",
    summary:"Генерация и разбор подписанных HMAC-токенов аутентификации с проверкой срока действия для Bearer-авторизации.",
    tags:["security","authentication","token","utility"], complexity:"moderate",
    languageNotes:"Подпись токенов через HMAC обеспечивает целостность без обращения к БД при проверке." },
];

// Build edges
const edges = [];
// 1:1 import edges
for (const src of Object.keys(imp)) {
  for (const tgt of imp[src]) {
    edges.push({ source:`file:${src}`, target:`file:${tgt}`, type:"imports", direction:"forward", weight:0.7 });
  }
}

// contains + exports edges for created function/class nodes
const containsMap = {
  "internal/deps/deps.go": [["class","AppDeps",true],["function","WithOrderBot",true],["function","WithAPIServerConfig",true]],
  "internal/inbound/api/httpx/apperror.go": [["function","WriteAppError",true]],
  "internal/inbound/api/httpx/auth.go": [["class","Authenticator",true],["class","MiniAppUser",true],["function","RequireMiniAppAuth",true],["function","RequireAdmin",true],["function","resolveUser",false],["function","buildDepartmentViewer",false],["function","validateMiniAppInitData",false]],
  "internal/inbound/api/httpx/respond.go": [["function","ParseRequestDate",true],["function","UniqueQueryValues",true]],
  "internal/inbound/api/middleware.go": [["function","cors",false],["function","recoverer",false]],
  "internal/inbound/api/server.go": [["class","Server",true],["function","NewServer",true],["function","Start",true]],
  "internal/inbound/bot/action_menu.go": [["class","actionMenuSnapshot",false],["function","actionMenu",false],["function","resolveState",false],["function","rows",false],["function","orderShopFilterReplyRows",false],["function","actionKeyboard",false],["function","currentUser",false],["function","userDepartmentType",false]],
  "internal/inbound/bot/bot.go": [["class","OrderBot",true],["function","NewOrderBot",true]],
  "internal/inbound/bot/handler_start.go": [["function","handleStart",false]],
  "internal/inbound/bot/handler_templates.go": [["function","handleTemplates",false],["function","handleTemplateAll",false],["function","handleTemplateUse",false]],
  "internal/inbound/bot/middleware.go": [["function","requirePermissions",false],["function","privateChatOnly",false],["function","authUserFromContext",false]],
  "internal/inbound/bot/mini_app.go": [["function","miniAppLink",false]],
  "internal/inbound/bot/sender.go": [["function","sendTelegramChunks",false],["function","splitTelegramMessage",false],["function","splitPlainText",false],["function","botSender",false]],
  "internal/pkg/apperr/apperr.go": [["class","Error",true]],
};
for (const f of Object.keys(containsMap)) {
  for (const [kind,name,exported] of containsMap[f]) {
    const nid = `${kind}:${f}:${name}`;
    edges.push({ source:`file:${f}`, target:nid, type:"contains", direction:"forward", weight:1.0 });
    if (exported) edges.push({ source:`file:${f}`, target:nid, type:"exports", direction:"forward", weight:0.8 });
  }
}

// tested_by edges (production -> test, cross-file within batch / known)
edges.push({ source:"file:internal/inbound/api/httpx/auth.go", target:"file:internal/inbound/api/httpx/auth_test.go", type:"tested_by", direction:"forward", weight:0.5 });
edges.push({ source:"file:internal/inbound/bot/action_menu.go", target:"file:internal/inbound/bot/action_menu_test.go", type:"tested_by", direction:"forward", weight:0.5 });
edges.push({ source:"file:internal/inbound/bot/mini_app.go", target:"file:internal/inbound/bot/mini_app_test.go", type:"tested_by", direction:"forward", weight:0.5 });
edges.push({ source:"file:internal/inbound/bot/sender.go", target:"file:internal/inbound/bot/sender_test.go", type:"tested_by", direction:"forward", weight:0.5 });
edges.push({ source:"file:internal/pkg/apperr/apperr.go", target:"file:internal/pkg/apperr/apperr_test.go", type:"tested_by", direction:"forward", weight:0.5 });

// cross-file calls (confident, within batch + neighborMap)
// apperror.go WriteAppError uses respond.go WriteError
edges.push({ source:"function:internal/inbound/api/httpx/apperror.go:WriteAppError", target:"function:internal/inbound/api/httpx/respond.go:UniqueQueryValues", type:"calls", direction:"forward", weight:0.8 });

const out = { nodes, edges };
fs.writeFileSync(".understand-anything/intermediate/batch-3.json", JSON.stringify(out, null, 2));
console.log("nodes:", nodes.length, "edges:", edges.length);
const impCount = edges.filter(e=>e.type==="imports").length;
console.log("import edges:", impCount);
// validate unique node ids
const ids = new Set(); let dup=0;
for (const n of nodes){ if(ids.has(n.id)) {dup++; console.log("DUP",n.id);} ids.add(n.id);}
console.log("dup ids:", dup);
// self edges
const self = edges.filter(e=>e.source===e.target).length;
console.log("self edges:", self);
