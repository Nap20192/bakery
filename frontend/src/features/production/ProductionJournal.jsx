import { useEffect, useState } from 'react';
import { Button } from '../../ui/Button';
import { CategoryBadge } from '../../ui/CategoryBadge';
import { ConfirmDialog } from '../../ui/ConfirmDialog';
import { EmptyState } from '../../ui/EmptyState';
import { ErrorBanner } from '../../ui/ErrorBanner';
import { SheetBadge } from '../../ui/SheetBadge';
import { panelClass, PanelHeader } from '../../ui/Panel';
import { MonitorReports } from '../baker/MonitorReports';
import { ProductionSheet } from './ProductionSheet';
import {
  deleteProductionSheet,
  fetchBatchOrderMonitor,
  fetchCategories,
  fetchOrder,
  fetchOrderMonitor,
  fetchProductionSheet,
  fetchProductionSheets,
  updateProductionSheet,
} from '../../api/orders';
import { formatDate, formatQuantity } from '../../lib/format';
import { buildProductionJournalMatrix, formatJournalDate } from './productionJournalModel';

// OrderLink — номер заказа как переход «отработка → заказ».
function OrderLink({ number, onOpenOrder }) {
  if (!onOpenOrder) {
    return <span className="min-w-0 truncate text-note leading-5 text-stone-500" title={number}>{number}</span>;
  }
  return (
    <button
      type="button"
      onClick={() => onOpenOrder(number)}
      title={`Открыть заказ ${number}`}
      className="min-w-0 truncate text-left text-note leading-5 text-stone-600 underline decoration-stone-300 underline-offset-2 hover:text-stone-900 focus:outline-none focus:ring-2 focus:ring-stone-900/20"
    >
      {number}
    </button>
  );
}

// ProductionJournal — журнал отработок: список документов и правка листа.
// Лист фиксирует партию (выбранные заказы) и отклонения факта; редактор —
// тот же ProductionSheet, что и при создании со страницы выбранных заказов,
// плюс расчёт теста по партии (факт учитывается автоматически — заказы
// приходят с декорированным produced_quantity).
// Связь: у отработки много заказов, у заказа — максимум одна отработка.
// initialSheetId открывает документ сразу (переход «заказ → отработка»),
// onOpenOrder ведёт обратно к заказу.
export function ProductionJournal({ initialSheetId = 0, onOpenOrder }) {
  const [sheets, setSheets] = useState([]);
  const [categories, setCategories] = useState([]);
  const [openSheet, setOpenSheet] = useState(null);
  const [sheetOrders, setSheetOrders] = useState([]);
  const [monitor, setMonitor] = useState(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [loadingList, setLoadingList] = useState(true);
  const [removing, setRemoving] = useState(null);
  // Констрейнт (docs/constraints.md): расчёт по партии — только при
  // сохранённом листе; несохранённые правки блокируют мониторинг.
  const [sheetDirty, setSheetDirty] = useState(false);

  async function load() {
    try {
      const [rows, categoryRows] = await Promise.all([fetchProductionSheets(), fetchCategories()]);
      const list = Array.isArray(rows) ? rows : [];
      const firstOrders = await Promise.allSettled(
        list.map((sheet) => sheet.order_numbers?.[0] ? fetchOrder(sheet.order_numbers[0]) : Promise.resolve(null)),
      );
      setSheets(list.map((sheet, index) => ({
        ...sheet,
        category: firstOrders[index]?.status === 'fulfilled' ? firstOrders[index].value?.category || null : null,
      })));
      setCategories(Array.isArray(categoryRows) ? categoryRows : []);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoadingList(false);
    }
  }

  useEffect(() => {
    const timer = window.setTimeout(() => {
      load();
      if (initialSheetId > 0) open(initialSheetId);
    }, 0);
    return () => window.clearTimeout(timer);
  }, [initialSheetId]);

  async function open(id) {
    setError('');
    setBusy(true);
    try {
      const sheet = await fetchProductionSheet(id);
      // Полные заказы партии — редактор и расчёт теста работают по ним.
      const orders = await Promise.all((sheet.order_numbers || []).map((number) => fetchOrder(number)));
      setOpenSheet(sheet);
      setSheetOrders(orders);
      setMonitor(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  function close() {
    setOpenSheet(null);
    setSheetOrders([]);
    setMonitor(null);
  }

  async function save(ordersPayload) {
    setError('');
    setBusy(true);
    try {
      const saved = await updateProductionSheet(openSheet.id, ordersPayload);
      // Перечитываем партию: заказы должны показать новый декорированный факт.
      const orders = await Promise.all((saved.order_numbers || []).map((number) => fetchOrder(number)));
      setOpenSheet(saved);
      setSheetOrders(orders);
      setMonitor(null);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  // Расчёт теста по партии листа: факт отработки уже учтён — мониторинг
  // считает по effective quantity декорированных заказов.
  async function calculate() {
    const numbers = openSheet?.order_numbers || [];
    if (!numbers.length) return;
    setError('');
    setBusy(true);
    try {
      const result = numbers.length === 1 ? await fetchOrderMonitor(numbers[0]) : await fetchBatchOrderMonitor(numbers);
      setMonitor(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function confirmRemove() {
    const sheet = removing;
    setRemoving(null);
    if (!sheet) return;
    setError('');
    setBusy(true);
    try {
      await deleteProductionSheet(sheet.id);
      if (openSheet?.id === sheet.id) close();
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  const journal = buildProductionJournalMatrix(sheets, categories);

  return (
    <section className="px-3 py-3 pb-20 sm:px-5 sm:pb-5 lg:px-6">
      <div className="mx-auto max-w-5xl space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h1 className="m-0 flex items-center gap-2 text-lg font-semibold">
            Отработки
            <span className="rounded-full bg-stone-100 px-2 py-0.5 text-note font-medium tabular-nums text-stone-600">{sheets.length}</span>
          </h1>
          <p className="m-0 text-note leading-5 text-stone-500">Новая отработка создаётся со страницы выбранных заказов.</p>
        </div>

        <ErrorBanner error={error} />

        {openSheet ? (
          <div className="space-y-4">
            <section className={panelClass}>
              <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                <div className="flex min-w-0 items-center gap-2">
                  <PanelHeader
                    title={`Отработка №${openSheet.id}`}
                    eyebrow={`${formatDate(openSheet.created_at)} · @${(openSheet.created_by_username || '').replace(/^@/, '')}`}
                  />
                  <SheetBadge sheetId={openSheet.id} />
                </div>
                <div className="flex shrink-0 gap-2">
                  <Button variant="danger" disabled={busy} onClick={() => setRemoving(openSheet)}>Удалить</Button>
                  <Button onClick={close}>К списку</Button>
                </div>
              </div>
              <div className="mb-3 flex flex-wrap items-center gap-x-3 gap-y-1 rounded-md bg-stone-50 px-2.5 py-1.5">
                <span className="text-caption font-medium uppercase text-stone-400">Заказы партии:</span>
                {(openSheet.order_numbers || []).map((number) => (
                  <OrderLink key={number} number={number} onOpenOrder={onOpenOrder} />
                ))}
              </div>
              <ProductionSheet
                key={`${openSheet.id}:${openSheet.updated_at}`}
                orders={sheetOrders}
                sheetItems={openSheet.items || []}
                loading={busy}
                onSave={save}
                onDirtyChange={setSheetDirty}
                submitLabel="Сохранить изменения"
              />
            </section>
            <section className={panelClass}>
              <PanelHeader title="Расчёт теста" eyebrow="С учётом отработки" />
              {sheetDirty && (
                <p className="m-0 mb-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-note text-amber-800" role="status">
                  Сначала сохраните или отмените правки отработки — расчёт идёт по сохранённому факту.
                </p>
              )}
              <MonitorReports
                monitor={sheetDirty ? null : monitor}
                onCalculate={calculate}
                loading={busy}
                canCalculate={(openSheet.order_numbers || []).length > 0 && !sheetDirty}
              />
            </section>
          </div>
        ) : loadingList ? (
          <p className="m-0 py-8 text-center text-body text-stone-500" role="status">Загружаем журнал…</p>
        ) : sheets.length ? (
          <ProductionJournalMatrix journal={journal} busy={busy} onOpen={open} onRemove={setRemoving} />
        ) : (
          <EmptyState>Отработок пока нет. Выберите заказы в матрице и сохраните факт выпечки.</EmptyState>
        )}
      </div>
      <ConfirmDialog
        open={Boolean(removing)}
        title={`Удалить отработку №${removing?.id || ''}?`}
        description="Заказы партии вернутся к факту «по заявке»."
        onConfirm={confirmRemove}
        onCancel={() => setRemoving(null)}
      />
    </section>
  );
}

function ProductionJournalMatrix({ journal, busy, onOpen, onRemove }) {
  return (
    <div className="overflow-auto rounded-xl border border-stone-200 bg-white shadow-sm">
      <table className="w-full min-w-[44rem] border-collapse text-body">
        <thead>
          <tr>
            <th className="sticky left-0 top-0 z-30 w-36 border-b border-r border-stone-200 bg-stone-50 px-3 py-2 text-left text-caption font-medium uppercase text-stone-500">
              Дата
            </th>
            {journal.columns.map((column) => (
              <th key={column.key} className="sticky top-0 z-20 min-w-60 border-b border-stone-200 bg-stone-50 px-3 py-2 text-left">
                {column.category ? <CategoryBadge category={column.category} /> : <span className="text-note font-semibold text-stone-600">Без типа</span>}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {journal.rows.map((row) => (
            <tr key={row.dateKey} className="border-b border-stone-100 last:border-0">
              <th className="sticky left-0 z-10 border-r border-stone-200 bg-white px-3 py-3 text-left align-top text-note font-semibold text-stone-700">
                {formatJournalDate(row.dateKey)}
              </th>
              {journal.columns.map((column) => {
                const cellSheets = row.cells[column.key] || [];
                return (
                  <td key={column.key} className="min-w-60 bg-white p-1.5 align-top">
                    {cellSheets.length ? (
                      <div className="space-y-1.5">
                        {cellSheets.map((sheet) => (
                          <ProductionJournalCell key={sheet.id} sheet={sheet} busy={busy} onOpen={onOpen} onRemove={onRemove} />
                        ))}
                      </div>
                    ) : (
                      <span className="block px-2 py-3 text-center text-stone-300" aria-label="Нет отработок">·</span>
                    )}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ProductionJournalCell({ sheet, busy, onOpen, onRemove }) {
  const orderCount = sheet.order_numbers?.length || 0;
  const deviationCount = Number(sheet.item_count) || 0;
  return (
    <div className="rounded-lg border border-stone-200 bg-stone-50/60 p-2">
      <button type="button" className="block w-full rounded-md text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-stone-900/30" onClick={() => onOpen(sheet.id)}>
        <span className="flex items-center gap-2 text-input font-semibold text-stone-950">
          <SheetBadge sheetId={sheet.id} deviations={deviationCount} />
          <span>Отработка №{sheet.id}</span>
        </span>
        <span className="mt-0.5 block text-note leading-5 text-stone-500">
          @{(sheet.created_by_username || '—').replace(/^@/, '')} · {orderCount} {orderCount === 1 ? 'заказ' : 'заказов'}
        </span>
        <span className={`block text-note leading-5 ${deviationCount ? 'font-medium text-amber-800' : 'text-stone-500'}`}>
          {deviationCount ? `Отклонений: ${formatQuantity(deviationCount)}` : 'Без отклонений'}
        </span>
      </button>
      <div className="mt-1.5 flex gap-1.5 border-t border-stone-200 pt-1.5">
        <Button className="flex-1" onClick={() => onOpen(sheet.id)}>Открыть</Button>
        <Button variant="danger" disabled={busy} onClick={() => onRemove(sheet)}>Удалить</Button>
      </div>
    </div>
  );
}
