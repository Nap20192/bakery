import { useEffect, useState } from 'react';
import { Button } from '../../ui/Button';
import { EmptyState } from '../../ui/EmptyState';
import { ErrorBanner } from '../../ui/ErrorBanner';
import { panelClass, PanelHeader } from '../../ui/Panel';
import { deleteProductionSheet, fetchProductionSheet, fetchProductionSheets, updateProductionSheet } from '../../api/orders';
import { formatDate, formatQuantity } from '../../lib/format';

const factInputClass =
  'h-9 w-20 rounded-md border border-stone-300 bg-white px-2 text-center text-sm tabular-nums outline-none focus:border-stone-900 focus:ring-2 focus:ring-stone-900/10';

// OrderLink — номер заказа как переход «отработка → заказ».
function OrderLink({ number, onOpenOrder }) {
  if (!onOpenOrder) {
    return <span className="min-w-0 truncate text-[12px] leading-5 text-stone-500" title={number}>{number}</span>;
  }
  return (
    <button
      type="button"
      onClick={() => onOpenOrder(number)}
      title={`Открыть заказ ${number}`}
      className="min-w-0 truncate text-left text-[12px] leading-5 text-stone-600 underline decoration-stone-300 underline-offset-2 hover:text-stone-900 focus:outline-none focus:ring-2 focus:ring-stone-900/20"
    >
      {number}
    </button>
  );
}

// ProductionJournal — журнал отработок: список документов, просмотр, правка
// значений факта и удаление. Факт в заказах пересчитывается на бэкенде.
// Связь: у отработки много заказов, у заказа — максимум одна отработка.
// initialSheetId открывает документ сразу (переход «заказ → отработка»),
// onOpenOrder ведёт обратно к заказу.
export function ProductionJournal({ initialSheetId = 0, onOpenOrder }) {
  const [sheets, setSheets] = useState([]);
  const [openSheet, setOpenSheet] = useState(null);
  const [drafts, setDrafts] = useState({});
  const [reasonDrafts, setReasonDrafts] = useState({});
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  function load() {
    fetchProductionSheets()
      .then((rows) => setSheets(Array.isArray(rows) ? rows : []))
      .catch((err) => setError(err instanceof Error ? err.message : String(err)));
  }

  useEffect(() => {
    load();
    if (initialSheetId > 0) open(initialSheetId);
  }, [initialSheetId]);

  async function open(id) {
    setError('');
    try {
      const sheet = await fetchProductionSheet(id);
      setOpenSheet(sheet);
      setDrafts({});
      setReasonDrafts({});
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  function draftValue(index, item) {
    const raw = drafts[index];
    return raw !== undefined ? raw : String(item.produced_quantity);
  }

  function reasonValue(index, item) {
    const raw = reasonDrafts[index];
    return raw !== undefined ? raw : (item.reason || '');
  }

  async function save() {
    setError('');
    setBusy(true);
    try {
      // Собираем позиции обратно в формат «заказ → позиции».
      const byOrder = new Map();
      openSheet.items.forEach((item, index) => {
        if (!byOrder.has(item.order_number)) byOrder.set(item.order_number, []);
        byOrder.get(item.order_number).push({
          product_name: item.product_name,
          produced_quantity: Number(draftValue(index, item) || 0),
          reason: reasonValue(index, item).trim(),
        });
      });
      const orders = [...byOrder.entries()].map(([number, items]) => ({ number, items }));
      const saved = await updateProductionSheet(openSheet.id, orders);
      // Пустой id — все значения сравнялись с заявкой и документ удалён.
      if (!saved?.id) {
        setOpenSheet(null);
      } else {
        setOpenSheet(saved);
      }
      setDrafts({});
      setReasonDrafts({});
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function remove(sheet) {
    if (!window.confirm(`Удалить отработку №${sheet.id}? Факт в заказах будет пересчитан.`)) return;
    setError('');
    setBusy(true);
    try {
      await deleteProductionSheet(sheet.id);
      if (openSheet?.id === sheet.id) setOpenSheet(null);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  const hasDraft = Object.keys(drafts).length > 0 || Object.keys(reasonDrafts).length > 0;

  return (
    <section className="px-3 py-3 pb-20 sm:px-5 sm:pb-5 lg:px-6">
      <div className="mx-auto max-w-5xl space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h1 className="m-0 flex items-center gap-2 text-lg font-semibold">
            Отработки
            <span className="rounded-full bg-stone-100 px-2 py-0.5 text-[12px] font-medium tabular-nums text-stone-600">{sheets.length}</span>
          </h1>
          <p className="m-0 text-[12px] leading-5 text-stone-500">Новая отработка создаётся со страницы выбранных заказов.</p>
        </div>

        <ErrorBanner error={error} />

        {openSheet ? (
          <section className={panelClass}>
            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
              <PanelHeader
                title={`Отработка №${openSheet.id}`}
                eyebrow={`${formatDate(openSheet.created_at)} · @${(openSheet.created_by_username || '').replace(/^@/, '')}`}
              />
              <Button onClick={() => setOpenSheet(null)}>К списку</Button>
            </div>
            <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,10rem)_5rem] items-center gap-2 px-1 pb-1 text-[11px] font-medium uppercase text-stone-400">
              <span>Продукт</span>
              <span>Заказ</span>
              <span className="text-center">Факт</span>
            </div>
            <div className="divide-y divide-stone-100">
              {openSheet.items.map((item, index) => (
                <div className="space-y-1 py-1.5" key={`${item.order_number}-${item.product_name}`}>
                  <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,10rem)_5rem] items-center gap-2">
                    <span className="min-w-0 break-words text-[13px] leading-5 text-stone-800">{item.product_name}</span>
                    <OrderLink number={item.order_number} onOpenOrder={onOpenOrder} />
                    <input
                      className={factInputClass}
                      type="text"
                      inputMode="numeric"
                      aria-label={`${item.product_name} ${item.order_number}: факт`}
                      value={draftValue(index, item)}
                      onChange={(event) => setDrafts((cur) => ({ ...cur, [index]: event.target.value.replace(/\D/g, '') }))}
                    />
                  </div>
                  <input
                    className="h-8 w-full rounded-md border border-stone-200 bg-stone-50/60 px-2 text-[13px] outline-none focus:border-stone-900 focus:bg-white focus:ring-2 focus:ring-stone-900/10"
                    type="text"
                    maxLength={200}
                    placeholder="Причина отклонения (необязательно)…"
                    aria-label={`${item.product_name} ${item.order_number}: причина`}
                    value={reasonValue(index, item)}
                    onChange={(event) => setReasonDrafts((cur) => ({ ...cur, [index]: event.target.value }))}
                  />
                </div>
              ))}
            </div>
            <div className="mt-3 flex flex-wrap items-center justify-between gap-2 border-t border-stone-200 pt-3">
              <p className="m-0 text-[12px] leading-5 text-stone-500">
                Отработка хранит только отклонения: значение, равное заявке, убирает строку;
                если совпадут все — документ удалится.
              </p>
              <div className="flex shrink-0 gap-2">
                <Button variant="danger" disabled={busy} onClick={() => remove(openSheet)}>Удалить отработку</Button>
                <Button variant="primary" disabled={busy || !hasDraft} onClick={save}>Сохранить изменения</Button>
              </div>
            </div>
          </section>
        ) : sheets.length ? (
          <div className="space-y-2">
            {sheets.map((sheet) => (
              <section className={`${panelClass} flex flex-wrap items-center justify-between gap-2`} key={sheet.id}>
                <button type="button" className="min-w-0 flex-1 text-left focus:outline-none" onClick={() => open(sheet.id)}>
                  <span className="block text-[14px] font-semibold leading-6 text-stone-950">
                    Отработка №{sheet.id}
                    <span className="ml-2 font-normal text-stone-500">{formatDate(sheet.created_at)}</span>
                  </span>
                  <span className="block truncate text-[12px] leading-5 text-stone-500">
                    @{(sheet.created_by_username || '—').replace(/^@/, '')} · заказов: {sheet.order_numbers.length} · позиций: {formatQuantity(sheet.item_count)}
                  </span>
                  <span className="block truncate text-[12px] leading-5 text-stone-400">{sheet.order_numbers.join(', ')}</span>
                </button>
                <div className="flex shrink-0 gap-2">
                  <Button onClick={() => open(sheet.id)}>Открыть</Button>
                  <Button variant="danger" disabled={busy} onClick={() => remove(sheet)}>Удалить</Button>
                </div>
              </section>
            ))}
          </div>
        ) : (
          <EmptyState>Отработок пока нет. Выберите заказы в матрице и сохраните факт выпечки.</EmptyState>
        )}
      </div>
    </section>
  );
}
