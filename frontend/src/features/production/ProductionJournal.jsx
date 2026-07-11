import { useEffect, useState } from 'react';
import { Button } from '../../ui/Button';
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
  fetchOrder,
  fetchOrderMonitor,
  fetchProductionSheet,
  fetchProductionSheets,
  updateProductionSheet,
} from '../../api/orders';
import { formatDate, formatQuantity } from '../../lib/format';

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
  const [openSheet, setOpenSheet] = useState(null);
  const [sheetOrders, setSheetOrders] = useState([]);
  const [monitor, setMonitor] = useState(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [removing, setRemoving] = useState(null);

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
                <span className="text-[11px] font-medium uppercase text-stone-400">Заказы партии:</span>
                {(openSheet.order_numbers || []).map((number) => (
                  <OrderLink key={number} number={number} onOpenOrder={onOpenOrder} />
                ))}
              </div>
              <ProductionSheet
                orders={sheetOrders}
                loading={busy}
                onSave={save}
                submitLabel="Сохранить изменения"
              />
            </section>
            <section className={panelClass}>
              <PanelHeader title="Расчёт теста" eyebrow="С учётом отработки" />
              <MonitorReports
                monitor={monitor}
                onCalculate={calculate}
                loading={busy}
                canCalculate={(openSheet.order_numbers || []).length > 0}
              />
            </section>
          </div>
        ) : sheets.length ? (
          <div className="space-y-2">
            {sheets.map((sheet) => (
              <section className={`${panelClass} flex flex-wrap items-center justify-between gap-2`} key={sheet.id}>
                <button type="button" className="min-w-0 flex-1 text-left focus:outline-none" onClick={() => open(sheet.id)}>
                  <span className="flex items-center gap-2 text-[14px] font-semibold leading-6 text-stone-950">
                    <SheetBadge sheetId={sheet.id} />
                    Отработка №{sheet.id}
                    <span className="font-normal text-stone-500">{formatDate(sheet.created_at)}</span>
                  </span>
                  <span className="block truncate text-[12px] leading-5 text-stone-500">
                    @{(sheet.created_by_username || '—').replace(/^@/, '')} · заказов: {sheet.order_numbers.length} · отклонений: {formatQuantity(sheet.item_count)}
                  </span>
                  <span className="block truncate text-[12px] leading-5 text-stone-400">{sheet.order_numbers.join(', ')}</span>
                </button>
                <div className="flex shrink-0 gap-2">
                  <Button onClick={() => open(sheet.id)}>Открыть</Button>
                  <Button variant="danger" disabled={busy} onClick={() => setRemoving(sheet)}>Удалить</Button>
                </div>
              </section>
            ))}
          </div>
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
