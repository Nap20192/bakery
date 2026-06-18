import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import TextField from '@mui/material/TextField';
import MuiButton from '@mui/material/Button';
import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { panelClass, PanelHeader } from '../../components/Panel';
import { formatDate, formatFulfillmentDate } from '../../lib/format';
import { orderSource } from '../../lib/orders';
import { MonitorReports } from './MonitorReports';
import { OrderDetails } from './OrderDetails';

function dateValue(offset = 0) {
  const date = new Date();
  date.setDate(date.getDate() + offset);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

export function BakerOrdersView({
  loading,
  orders,
  page,
  shops,
  filters,
  selectedNumber,
  selectedOrder,
  selectedOrderNumbers,
  monitor,
  error,
  onSelect,
  onToggleSelection,
  onPageChange,
  onFiltersChange,
  onResetFilters,
  onCalculate,
}) {
  const selectedCount = selectedOrderNumbers.length;

  return (
    <section className="px-3 py-3 sm:px-5 lg:px-6">
      <div className="mx-auto max-w-[1440px] space-y-4">
        <BakerOrderFilters
          filters={filters}
          shops={shops}
          loading={loading}
          selectedCount={selectedCount}
          onFiltersChange={onFiltersChange}
          onResetFilters={onResetFilters}
          onCalculate={onCalculate}
        />

        {error && <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] text-red-800">{error}</div>}

        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {orders.length ? (
            orders.map((order) => (
              <BakerOrderCard
                key={order.number}
                order={order}
                selected={selectedNumber === order.number}
                checked={selectedOrderNumbers.includes(order.number)}
                loading={loading}
                onSelect={() => onSelect(order.number)}
                onToggleSelection={() => onToggleSelection(order.number)}
              />
            ))
          ) : (
            <div className="sm:col-span-2 xl:col-span-4">
              <EmptyState compact>Заказов нет.</EmptyState>
            </div>
          )}
        </div>

        <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-2">
          <Button onClick={() => onPageChange(page.page - 1)} disabled={loading || page.page <= 1}>
            Назад
          </Button>
          <span className="whitespace-nowrap text-xs text-stone-500">
            {page.page} / {page.total_pages || 1}
          </span>
          <Button onClick={() => onPageChange(page.page + 1)} disabled={loading || page.page >= (page.total_pages || 1)}>
            Далее
          </Button>
        </div>

        {selectedOrder ? (
          <div className="grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_minmax(22rem,0.85fr)]">
            <section className={panelClass}>
              <OrderDetails order={selectedOrder} />
            </section>
            <section className={panelClass}>
              <PanelHeader title="Расчёт теста" />
              {selectedCount > 0 && (
                <div className="mb-3 flex flex-wrap items-center gap-2">
                  <Button variant="primary" onClick={onCalculate} disabled={loading}>
                    Рассчитать выбранные
                  </Button>
                  <span className="text-[13px] leading-5 text-stone-600">Выбрано заказов: {selectedCount}</span>
                </div>
              )}
              <MonitorReports monitor={monitor} />
            </section>
          </div>
        ) : (
          <EmptyState>Выберите заказ для просмотра.</EmptyState>
        )}
      </div>
    </section>
  );
}

function BakerOrderFilters({ filters, shops, loading, selectedCount, onFiltersChange, onResetFilters, onCalculate }) {
  return (
    <section className="rounded-lg border border-stone-300 bg-[#fff7df] p-3">
      <div className="grid gap-2 md:grid-cols-[minmax(12rem,1fr)_minmax(11rem,0.8fr)_auto] md:items-end">
        <FormControl size="small" fullWidth>
          <InputLabel id="baker-shop-filter-label">Магазин</InputLabel>
          <Select
            labelId="baker-shop-filter-label"
            label="Магазин"
            value={filters.fromDepartmentID || ''}
            onChange={(event) => onFiltersChange({ fromDepartmentID: event.target.value })}
          >
            <MenuItem value="">Все магазины</MenuItem>
            {shops.map((shop) => (
              <MenuItem value={String(shop.id)} key={shop.id}>
                {shop.name}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <TextField
          size="small"
          label="Дата выполнения"
          type="date"
          value={filters.fulfillmentDate || ''}
          onChange={(event) => onFiltersChange({ fulfillmentDate: event.target.value })}
          slotProps={{ inputLabel: { shrink: true } }}
          fullWidth
        />

        <div className="grid grid-cols-4 gap-1.5 md:min-w-[25rem]">
          <MuiButton size="small" variant="outlined" onClick={() => onFiltersChange({ fulfillmentDate: dateValue() })}>
            Сегодня
          </MuiButton>
          <MuiButton size="small" variant="outlined" onClick={() => onFiltersChange({ fulfillmentDate: dateValue(1) })}>
            Завтра
          </MuiButton>
          <MuiButton size="small" variant="text" onClick={onResetFilters}>
            Сброс
          </MuiButton>
          <MuiButton size="small" variant="contained" disabled={loading || selectedCount === 0} onClick={onCalculate}>
            Расчёт
          </MuiButton>
        </div>
      </div>
    </section>
  );
}

function BakerOrderCard({ order, selected, checked, loading, onSelect, onToggleSelection }) {
  return (
    <article
      className={`rounded-lg border bg-[#fff7df] p-3 transition ${
        selected ? 'border-stone-950 shadow-sm' : 'border-stone-300 hover:border-stone-500'
      }`}
    >
      <div className="mb-2 flex items-start justify-between gap-3">
        <button type="button" className="min-w-0 flex-1 text-left" onClick={onSelect}>
          <strong className="block truncate text-[16px] font-semibold leading-6 text-stone-950">{order.number}</strong>
          <span className="block truncate text-[12px] leading-5 text-stone-600">{orderSource(order)}</span>
        </button>
        <label className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-stone-300 bg-[#fff1cb]">
          <input
            type="checkbox"
            className="h-4 w-4 accent-stone-950"
            checked={checked}
            disabled={loading}
            aria-label={`Выбрать заказ ${order.number}`}
            onChange={onToggleSelection}
          />
        </label>
      </div>

      <button type="button" className="grid w-full grid-cols-2 gap-2 text-left" onClick={onSelect}>
        <CardMeta label="Выполнить" value={formatFulfillmentDate(order.fulfillment_date) || '-'} strong />
        <CardMeta label="Позиций" value={String(order.items?.length || 0)} />
        <CardMeta label="Создан" value={formatDate(order.created_at) || '-'} />
        <CardMeta label="Куда" value={order.to_department?.name || '-'} />
      </button>
    </article>
  );
}

function CardMeta({ label, value, strong = false }) {
  return (
    <span className="min-w-0 rounded-md border border-stone-300 bg-[#fff1cb] px-2 py-1.5">
      <span className="block text-[10px] font-medium uppercase leading-4 text-stone-500">{label}</span>
      <span className={`block truncate text-[13px] leading-5 text-stone-900 ${strong ? 'font-semibold' : 'font-medium'}`}>{value}</span>
    </span>
  );
}
