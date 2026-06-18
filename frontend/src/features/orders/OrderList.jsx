import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import MuiButton from '@mui/material/Button';
import { Button } from '../../components/Button';
import { EmptyState } from '../../components/EmptyState';
import { formatFulfillmentDate } from '../../lib/format';
import { orderSource } from '../../lib/orders';

function todayValue(offset = 0) {
  const date = new Date();
  date.setDate(date.getDate() + offset);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

export function OrderList({
  loading,
  orders,
  page,
  shops,
  viewer,
  canFilterShops,
  filters,
  selectedNumber,
  onSelect,
  onPageChange,
  onFiltersChange,
  onResetFilters,
}) {
  return (
    <aside className="m-3 flex min-h-[32rem] max-h-[70vh] flex-col gap-2.5 rounded-lg border border-stone-300 bg-white p-3 shadow-sm lg:sticky lg:top-[3.75rem] lg:ml-3 lg:mr-0 lg:h-[calc(100vh-5.25rem)] lg:max-h-none lg:min-h-[38rem]">
      <div className="flex items-center justify-between gap-2">
        <div className="min-w-0">
          <h1 className="m-0 flex items-center gap-2 text-[20px] font-semibold leading-7 text-stone-950">
            Заказы
            {page.total > 0 && (
              <span className="rounded-full bg-stone-100 px-2 py-0.5 text-[12px] font-medium tabular-nums text-stone-600">{page.total}</span>
            )}
          </h1>
          {viewer?.department_name && (
            <p className="m-0 truncate text-[12px] leading-5 text-stone-600">
              {viewer.department_name}
              {viewer.telegram_username && ` / @${viewer.telegram_username}`}
            </p>
          )}
        </div>
      </div>

      <div className="rounded-md border border-stone-200 bg-stone-50 p-2">
        <Stack spacing={1.25}>
          {canFilterShops && (
            <FormControl size="small" fullWidth>
              <InputLabel id="shop-filter-label">Магазин</InputLabel>
              <Select
                labelId="shop-filter-label"
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
          )}

          <TextField
            size="small"
            label="Дата выполнения"
            type="date"
            value={filters.fulfillmentDate || ''}
            onChange={(event) => onFiltersChange({ fulfillmentDate: event.target.value })}
            slotProps={{ inputLabel: { shrink: true } }}
            fullWidth
          />

          <div className="grid grid-cols-3 gap-1.5">
            <MuiButton size="small" variant="outlined" onClick={() => onFiltersChange({ fulfillmentDate: todayValue() })}>
              Сегодня
            </MuiButton>
            <MuiButton size="small" variant="outlined" onClick={() => onFiltersChange({ fulfillmentDate: todayValue(1) })}>
              Завтра
            </MuiButton>
            <MuiButton size="small" variant="text" onClick={onResetFilters}>
              Сброс
            </MuiButton>
          </div>
        </Stack>
      </div>

      <div className="min-h-0 flex-1 space-y-1 overflow-y-auto pr-1">
        {orders.length ? (
          orders.map((order) => {
            return (
              <div
                key={order.number}
                className={`w-full rounded-md border px-2.5 py-2 text-left transition ${
                  selectedNumber === order.number ? 'border-stone-300 bg-stone-50' : 'border-transparent bg-white hover:border-stone-300 hover:bg-stone-50'
                }`}
              >
                <div className="grid grid-cols-1 gap-2">
                  <button className="min-w-0 text-left" onClick={() => onSelect(order.number)}>
                    <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-3 gap-y-0.5">
                      <strong className="break-words text-[13px] font-semibold leading-5 text-stone-900">{order.number}</strong>
                      <span className="text-[12px] leading-5 text-stone-600">{order.items?.length || 0} поз.</span>
                      <strong className="break-words text-[12px] font-semibold leading-5 text-stone-800">Откуда: {orderSource(order)}</strong>
                      <span className="text-[12px] leading-5 text-stone-600">{formatFulfillmentDate(order.fulfillment_date) || '-'}</span>
                    </div>
                  </button>
                </div>
              </div>
            );
          })
        ) : (
          <EmptyState compact>Заказов нет.</EmptyState>
        )}
      </div>

      <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-2">
        <Button onClick={() => onPageChange(page.page - 1)} disabled={loading || page.page <= 1}>
          Назад
        </Button>
        <span className="whitespace-nowrap text-center text-xs text-stone-500 tabular-nums">
          {page.page} / {page.total_pages || 1}
        </span>
        <Button onClick={() => onPageChange(page.page + 1)} disabled={loading || page.page >= (page.total_pages || 1)}>
          Далее
        </Button>
      </div>
    </aside>
  );
}
