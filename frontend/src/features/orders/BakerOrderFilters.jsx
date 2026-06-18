import { Button } from '../../components/Button';
import { InputField, SelectField } from '../../components/Field';
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
        <SelectField
          label="Магазин"
          value={filters.fromDepartmentID || ''}
          onChange={(event) => onFiltersChange({ fromDepartmentID: event.target.value })}
        >
          <option value="">Все магазины</option>
          {shops.map((shop) => (
            <option value={String(shop.id)} key={shop.id}>
              {shop.name}
            </option>
          ))}
        </SelectField>

        <InputField
          label="Дата выполнения"
          type="date"
          value={filters.fulfillmentDate || ''}
          onChange={(event) => onFiltersChange({ fulfillmentDate: event.target.value })}
        />

        <div className="grid grid-cols-2 gap-1.5 sm:grid-cols-4 md:min-w-[25rem]">
          <Button onClick={() => onFiltersChange({ fulfillmentDate: dateValue() })}>Сегодня</Button>
          <Button onClick={() => onFiltersChange({ fulfillmentDate: dateValue(1) })}>Завтра</Button>
          <Button onClick={onResetFilters}>Сброс</Button>
          <Button variant={selectionMode ? 'primary' : 'default'} onClick={onToggleSelectionMode}>
            <Icon name="select" size={15} />
            Выбор
          </Button>
        </div>
      </div>
    </section>
  );
}
