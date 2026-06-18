import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import TextField from '@mui/material/TextField';
import MuiButton from '@mui/material/Button';
import { Icon } from '../../components/Icon';

function dateValue(offset = 0) {
  const date = new Date();
  date.setDate(date.getDate() + offset);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

export function BakerOrderFilters({ filters, shops, selectionMode, onToggleSelectionMode, onFiltersChange, onResetFilters }) {
  return (
    <section className="rounded-lg border border-stone-300 bg-white p-3 shadow-sm">
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

        <div className="grid grid-cols-2 gap-1.5 sm:grid-cols-4 md:min-w-[25rem]">
          <MuiButton size="small" variant="outlined" onClick={() => onFiltersChange({ fulfillmentDate: dateValue() })}>
            Сегодня
          </MuiButton>
          <MuiButton size="small" variant="outlined" onClick={() => onFiltersChange({ fulfillmentDate: dateValue(1) })}>
            Завтра
          </MuiButton>
          <MuiButton size="small" variant="text" onClick={onResetFilters}>
            Сброс
          </MuiButton>
          <button
            type="button"
            onClick={onToggleSelectionMode}
            className={`rounded-md border px-2 py-1.5 text-[12px] font-semibold transition ${
              selectionMode
                ? 'border-stone-950 bg-stone-950 text-white'
                : 'border-stone-400 bg-stone-50 text-stone-900 hover:border-stone-700'
            }`}
          >
            <Icon name="select" size={15} className="mr-1 inline-block align-[-3px]" />
            Выбор нескольких
          </button>
        </div>
      </div>
    </section>
  );
}
